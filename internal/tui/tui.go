package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hammie/rubrduck/internal/config"
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	inputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205"))

	messageStyle = lipgloss.NewStyle().
			PaddingLeft(1)

	userMessageStyle = messageStyle.Copy().
				Foreground(lipgloss.Color("170"))

	aiMessageStyle = messageStyle.Copy().
			Foreground(lipgloss.Color("86"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)

type model struct {
	config   *config.Config
	messages []message
	input    string
	width    int
	height   int
	ready    bool
	quitting bool
}

type message struct {
	role    string // "user" or "assistant"
	content string
}

func initialModel(cfg *config.Config) model {
	return model{
		config:   cfg,
		messages: []message{},
		input:    "",
	}
}

func (m model) Init() tea.Cmd {
	return tea.SetWindowTitle("RubrDuck")
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.quitting = true
			return m, tea.Quit
		case tea.KeyEnter:
			if m.input != "" {
				// Add user message
				m.messages = append(m.messages, message{
					role:    "user",
					content: m.input,
				})

				// TODO: Send to AI and get response
				// For now, just echo back
				m.messages = append(m.messages, message{
					role:    "assistant",
					content: fmt.Sprintf("I received: %s\n\n(AI integration coming soon!)", m.input),
				})

				m.input = ""
			}
		case tea.KeyBackspace:
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		default:
			m.input += msg.String()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.ready {
			m.ready = true
		}
	}

	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	if !m.ready {
		return "Initializing..."
	}

	var s strings.Builder

	// Title
	title := titleStyle.Render("🦆 RubrDuck - AI Coding Assistant")
	s.WriteString(title + "\n\n")

	// Messages
	for _, msg := range m.messages {
		var style lipgloss.Style
		var prefix string

		if msg.role == "user" {
			style = userMessageStyle
			prefix = "You: "
		} else {
			style = aiMessageStyle
			prefix = "RubrDuck: "
		}

		s.WriteString(style.Render(prefix+msg.content) + "\n\n")
	}

	// Input prompt
	prompt := promptStyle.Render("> ") + inputStyle.Render(m.input)
	s.WriteString(prompt)

	// Help text
	helpText := helpStyle.Render("\n\nPress ESC or Ctrl+C to quit")

	// Calculate remaining space and add help at bottom
	content := s.String()
	lines := strings.Count(content, "\n")
	remainingLines := m.height - lines - 2

	if remainingLines > 0 {
		s.WriteString(strings.Repeat("\n", remainingLines))
	}
	s.WriteString(helpText)

	return s.String()
}

// Run starts the TUI application
func Run(cfg *config.Config) error {
	p := tea.NewProgram(initialModel(cfg), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
