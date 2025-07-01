package tui

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hammie/rubrduck/internal/config"
)

// Response represents a single AI response in the conversation
type Response struct {
	Mode   ViewMode
	Query  string
	Answer string
	Time   string
}

// Model is the root TUI model managing modes, input, and global state
type Model struct {
	config         *config.Config
	width, height  int
	viewMode       ViewMode
	selectedOption int

	input   textinput.Model
	focused bool

	quitting     bool
	responses    []Response
	scrollOffset int
}

// NewModel initializes a new TUI model using provided configuration
func NewModel(cfg *config.Config) Model {
	m := Model{
		config:         cfg,
		selectedOption: 0,
		responses:      make([]Response, 0),
		scrollOffset:   0,
		focused:        false,
	}

	// Initialize text input
	ti := textinput.New()
	ti.Placeholder = "Type your question or command..."
	ti.Prompt = "💬 "
	ti.CharLimit = 0
	m.input = ti

	// Determine initial mode from config or prompt selection
	switch s := strings.ToLower(cfg.TUI.StartMode); s {
	case "planning":
		m.viewMode = ViewModePlanning
	case "building":
		m.viewMode = ViewModeBuilding
	case "debugging":
		m.viewMode = ViewModeDebugging
	case "tech-debt", "tech debt":
		m.viewMode = ViewModeTechDebt
	default:
		m.viewMode = ViewModeSelect
	}
	return m
}

// Init is called when the TUI starts; it sets up the alternate screen
func (m Model) Init() tea.Cmd {
	return tea.EnterAltScreen
}

// Update processes messages and handles global and mode-specific events
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle window resizing
	if win, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = win.Width
		m.height = win.Height
		m.input.Width = win.Width - 10
	}

	// Global quit
	if key, ok := msg.(tea.KeyMsg); ok && key.Type == tea.KeyCtrlC {
		m.quitting = true
		return m, tea.Quit
	}

	if m.viewMode == ViewModeSelect {
		m, _ = m.updateModeSelect(msg)
		if key, ok := msg.(tea.KeyMsg); ok && key.Type == tea.KeyEnter {
			m.focused = true
			cmds = append(cmds, m.input.Focus())
		}
	} else {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.Type {
			case tea.KeyTab:
				m.focused = !m.focused
				if m.focused {
					cmds = append(cmds, m.input.Focus())
				} else {
					m.input.Blur()
				}
			case tea.KeyEscape:
				m.viewMode = ViewModeSelect
			}
		case tea.MouseMsg:
			// Click/tap anywhere refocuses the input
			if msg.Button != tea.MouseButtonWheelUp && msg.Button != tea.MouseButtonWheelDown {
				m.focused = true
				m.input.Focus()
			} else if !m.focused {
				// Scroll with mouse wheel when not focused
				if msg.Button == tea.MouseButtonWheelUp && m.scrollOffset > 0 {
					m.scrollOffset--
				}
				if msg.Button == tea.MouseButtonWheelDown {
					m.scrollOffset++
				}
			}
		}

		if m.focused {
			m.input, _ = m.input.Update(msg)
			if key, ok := msg.(tea.KeyMsg); ok && key.Type == tea.KeyEnter {
				q := strings.TrimSpace(m.input.Value())
				if q != "" {
					m.responses = append(m.responses, Response{Mode: m.viewMode, Query: q, Time: "Just now"})
					m.responses = append(m.responses, m.generateSimulatedResponse(q))
					m.input.Reset()
					cmds = append(cmds, m.input.Focus())
					m.adjustScrollToBottom()
				}
			}
		} else {
			if key, ok := msg.(tea.KeyMsg); ok {
				switch key.Type {
				case tea.KeyUp:
					if m.scrollOffset > 0 {
						m.scrollOffset--
					}
				case tea.KeyDown:
					if m.scrollOffset < len(m.responses)-1 {
						m.scrollOffset++
					}
				}
			}
		}
	}

	m.clampScroll()
	return m, tea.Batch(cmds...)
}

// View renders the active mode screen along with input and help bars
func (m Model) View() string {
	if m.quitting {
		return "Thanks for using RubrDuck! 🦆\n"
	}

	// Reserve lines for input (5) + status bar (1)
	contentHeight := m.height - 6
	if contentHeight < 3 {
		contentHeight = 3
	}

	var body string
	if m.viewMode == ViewModeSelect && len(m.responses) == 0 {
		body = m.renderModeSelect()
	} else if len(m.responses) > 0 {
		body = m.renderConversation(contentHeight)
	} else {
		switch m.viewMode {
		case ViewModePlanning:
			body = m.renderPlanning()
		case ViewModeBuilding:
			body = m.renderBuilding()
		case ViewModeDebugging:
			body = m.renderDebugging()
		case ViewModeTechDebt:
			body = m.renderTechDebt()
		}
	}

	content := lipgloss.NewStyle().Height(contentHeight).Width(m.width - 2).Padding(1).Render(body)

	var inputView string
	if m.viewMode != ViewModeSelect {
		inputView = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(primaryColor).Padding(1).Width(m.width - 4).Render(m.input.View())
	}

	statusView := renderStatusBar(m)
	return lipgloss.JoinVertical(lipgloss.Left, content, inputView, statusView)
}

// Run launches the TUI application
func Run(cfg *config.Config) error {
	p := tea.NewProgram(NewModel(cfg), tea.WithContext(context.Background()), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}

// generateSimulatedResponse creates a fake AI response for testing
func (m Model) generateSimulatedResponse(query string) Response {
	templates := map[ViewMode]string{
		ViewModePlanning:  "Based on your request, here's a comprehensive plan for: " + query,
		ViewModeBuilding:  "Here's a basic implementation plan for: " + query,
		ViewModeDebugging: "Debug steps for your issue: " + query,
		ViewModeTechDebt:  "Tech debt assessment for: " + query,
	}
	answer, ok := templates[m.viewMode]
	if !ok {
		answer = "I understand your request. Let me help you with that."
	}
	return Response{Mode: m.viewMode, Query: query, Answer: answer, Time: "Just now"}
}

// SetMode changes the current view mode
func (m *Model) SetMode(mode ViewMode) {
	m.viewMode = mode
}

// SetSize updates the model's dimensions and adjusts input width
func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.input.Width = w - 10
}

// SetFocused toggles whether the input is focused
func (m *Model) SetFocused(f bool) {
	m.focused = f
	if f {
		m.input.Focus()
	} else {
		m.input.Blur()
	}
}

// clampScroll ensures scrollOffset is not negative or beyond the available responses.
func (m *Model) clampScroll() {
	max := len(m.responses) - 1
	if max < 0 {
		max = 0
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	} else if m.scrollOffset > max {
		m.scrollOffset = max
	}
}

// adjustScrollToBottom positions the view so that the last response is visible
func (m *Model) adjustScrollToBottom() {
	// Scroll so that the newest responses fill the content window
	visible := m.height - 12
	if visible < 1 {
		visible = 1
	}
	m.scrollOffset = len(m.responses) - visible
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

// renderConversation shows the chat history with proper scrolling
func (m Model) renderConversation(maxHeight int) string {
	if len(m.responses) == 0 {
		return m.renderTitle() + "\n\nNo conversation yet. Type a message below to get started!"
	}

	var parts []string
	parts = append(parts, m.renderTitle(), "")

	visibles := m.responses[m.scrollOffset:]
	for i, resp := range visibles {
		if len(parts) > maxHeight-5 {
			break
		}
		if resp.Answer == "" {
			userStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00D7FF")).Bold(true)
			parts = append(parts, userStyle.Render("👤 You: ")+resp.Query)
		} else {
			asstStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED")).Bold(true)
			parts = append(parts, asstStyle.Render("🤖 Assistant: "))

			wrapped := m.wrapText(resp.Answer, m.width-6)
			parts = append(parts, wrapped)
		}
		if i < len(visibles)-1 {
			parts = append(parts, "")
		}
	}
	return strings.Join(parts, "\n")
}

// wrapText wraps text to fit within the specified width
func (m Model) wrapText(text string, width int) string {
	if width <= 0 {
		width = 80
	}
	lines := strings.Split(text, "\n")
	var wrapped []string
	for _, line := range lines {
		if len(line) <= width {
			wrapped = append(wrapped, line)
			continue
		}
		words := strings.Fields(line)
		current := words[0]
		for _, w := range words[1:] {
			if len(current)+1+len(w) <= width {
				current += " " + w
			} else {
				wrapped = append(wrapped, current)
				current = w
			}
		}
		wrapped = append(wrapped, current)
	}
	return strings.Join(wrapped, "\n")
}

// renderTitle displays the common title header
func (m Model) renderTitle() string {
	return lipgloss.NewStyle().Bold(true).Foreground(primaryColor).Render("🦆 RubrDuck - " + ModeName(m.viewMode))
}
