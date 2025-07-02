package tui2

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/hammie/rubrduck/internal/config"
)

// Run starts the Bubble Tea program for the interactive TUI.
func Run(cfg *config.Config) error {
	p := tea.NewProgram(
		newModel(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	_, err := p.Run()
	return err
}

type message struct {
	sender string // "user" or "ai"
	text   string
}

type model struct {
	spinner       spinner.Model
	viewport      viewport.Model
	input         textinput.Model
	messages      []message
	loading       bool
	userScrolling bool // tracks if user is manually scrolling
}

// newModel initializes the TUI model with default components.
func newModel() model {
	// Spinner for AI thinking indicator
	s := spinner.New()
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	// Viewport for chat history
	vp := viewport.New(0, 0)

	// Text input for user prompt
	ti := textinput.New()
	ti.Placeholder = "Ask AI..."
	ti.Focus()
	ti.Prompt = "❯ "
	ti.CharLimit = 256

	return model{
		spinner:       s,
		viewport:      vp,
		input:         ti,
		messages:      make([]message, 0),
		loading:       false,
		userScrolling: false,
	}
}

// Init is the initial command for Bubble Tea.
func (m model) Init() tea.Cmd {
	return spinner.Tick
}

// Update handles incoming messages and updates the model state.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyUp:
			// Allow scrolling up in the viewport
			m.viewport.LineUp(1)
			m.userScrolling = true
			return m, nil
		case tea.KeyDown:
			// Allow scrolling down in the viewport
			m.viewport.LineDown(1)
			// Check if we scrolled back to the bottom
			if m.viewport.AtBottom() {
				m.userScrolling = false
			}
			return m, nil
		case tea.KeyEnter:
			if m.input.Value() != "" {
				userText := m.input.Value()
				m.messages = append(m.messages, message{sender: "user", text: userText})
				// Render and scroll viewport to bottom
				content := lipgloss.NewStyle().Width(m.viewport.Width).Render(renderMessages(m.messages))
				m.viewport.SetContent(content)
				m.viewport.GotoBottom()
				m.userScrolling = false // Reset scrolling state when sending message
				m.input.Reset()
				m.loading = true
				return m, tea.Batch(
					spinner.Tick,
					waitForResponse(userText),
				)
			}
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case respondMsg:
		m.loading = false
		m.messages = append(m.messages, message{sender: "ai", text: msg.response})
		// Render messages with new content
		content := lipgloss.NewStyle().Width(m.viewport.Width).Render(renderMessages(m.messages))
		m.viewport.SetContent(content)
		// Only scroll to bottom if user wasn't manually scrolling
		if !m.userScrolling {
			m.viewport.GotoBottom()
		}
		return m, nil

	case tea.WindowSizeMsg:
		width := msg.Width
		height := msg.Height
		// Resize viewport to fill remaining space after input
		m.viewport.Width = width
		// Reserve 1 line for input
		m.viewport.Height = height - 1
		// Resize input to full width
		m.input.Width = width
		// Re-render messages with new width
		if len(m.messages) > 0 {
			content := lipgloss.NewStyle().Width(width).Render(renderMessages(m.messages))
			m.viewport.SetContent(content)
			// Only scroll to bottom if user wasn't manually scrolling
			if !m.userScrolling {
				m.viewport.GotoBottom()
			}
		}
		return m, nil
	}

	// Always update text input (allow typing even when loading)
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// View renders the TUI.
func (m model) View() string {
	var inputView string
	if m.loading {
		// Show spinner with input field when AI is thinking
		spinner := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render(m.spinner.View() + " AI thinking... ")
		inputView = lipgloss.JoinHorizontal(lipgloss.Left, spinner, m.input.View())
	} else {
		inputView = m.input.View()
	}
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.viewport.View(),
		lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(inputView),
	)
}

// renderMessages formats the chat history for the viewport.
func renderMessages(msgs []message) string {
	var out string
	for _, m := range msgs {
		var prefix string
		if m.sender == "user" {
			prefix = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render("You:   ")
		} else {
			prefix = lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Render("AI:    ")
		}
		out += prefix + m.text + "\n"
	}
	return out
}

// respondMsg carries a simulated AI response.
type respondMsg struct {
	response string
}

// waitForResponse simulates generating an AI response after a short delay.
func waitForResponse(input string) tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		// Placeholder echo response; replace with actual AI call integration
		return respondMsg{response: "Echo: " + input}
	})
}
