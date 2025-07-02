package tui2

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/hammie/rubrduck/internal/agent"
	_ "github.com/hammie/rubrduck/internal/ai/providers" // Register AI providers
	"github.com/hammie/rubrduck/internal/config"
)

// ViewMode represents the different TUI modes
type ViewMode int

const (
	ViewModeSelect ViewMode = iota
	ViewModePlanning
	ViewModeBuilding
	ViewModeDebugging
	ViewModeEnhance
)

// Mode info for selection and display
type ModeInfo struct {
	Name        string
	Description string
	Icon        string
	Mode        ViewMode
	Welcome     string
	Prompt      string
}

var modes = []ModeInfo{
	{
		Name:        "Planning",
		Description: "Architecture design and project planning",
		Icon:        "📋",
		Mode:        ViewModePlanning,
		Welcome:     "Planning Mode - Let's design your system architecture and break down complex projects into manageable tasks.",
		Prompt:      "What would you like to plan?",
	},
	{
		Name:        "Building",
		Description: "Code implementation and development",
		Icon:        "🔨",
		Mode:        ViewModeBuilding,
		Welcome:     "Building Mode - Time to implement features, generate code, and bring your ideas to life.",
		Prompt:      "What would you like to build?",
	},
	{
		Name:        "Debugging",
		Description: "Error analysis and problem solving",
		Icon:        "🐛",
		Mode:        ViewModeDebugging,
		Welcome:     "Debugging Mode - Let's analyze errors, trace issues, and solve those tricky problems.",
		Prompt:      "What issue are you debugging?",
	},
	{
		Name:        "Enhance",
		Description: "Code quality improvement and refactoring",
		Icon:        "🔧",
		Mode:        ViewModeEnhance,
		Welcome:     "Enhance Mode - Improve code quality, refactor legacy code, and modernize your codebase.",
		Prompt:      "What would you like to enhance?",
	},
}

// Run starts the Bubble Tea program for the interactive TUI.
func Run(cfg *config.Config) error {
	// Initialize AI agent with tools
	agent, err := agent.New(cfg)
	if err != nil {
		return err
	}

	p := tea.NewProgram(
		newModel(cfg, agent),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	_, err = p.Run()
	return err
}

type message struct {
	sender string // "user" or "ai"
	text   string
	mode   ViewMode
}

type model struct {
	// UI components
	spinner  spinner.Model
	viewport viewport.Model
	input    textinput.Model

	// State
	viewMode       ViewMode
	selectedOption int
	messages       []message
	loading        bool
	userScrolling  bool

	// Dimensions
	width  int
	height int

	// AI integration
	config *config.Config
	agent  *agent.Agent
}

// newModel initializes the TUI model with default components.
func newModel(cfg *config.Config, agent *agent.Agent) model {
	// Spinner for AI thinking indicator
	s := spinner.New()
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	// Viewport for chat history or mode selection
	vp := viewport.New(0, 0)

	// Text input for user prompt
	ti := textinput.New()
	ti.Placeholder = "Type your message..."
	ti.Focus()
	ti.Prompt = "❯ "
	ti.CharLimit = 500

	return model{
		spinner:        s,
		viewport:       vp,
		input:          ti,
		viewMode:       ViewModeSelect,
		selectedOption: 0,
		messages:       make([]message, 0),
		loading:        false,
		userScrolling:  false,
		config:         cfg,
		agent:          agent,
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
		if m.viewMode == ViewModeSelect {
			return m.updateModeSelect(msg)
		}
		return m.updateChatMode(msg)

	case tea.MouseMsg:
		if m.viewMode == ViewModeSelect {
			return m.updateModeSelectMouse(msg)
		}
		return m.updateChatModeMouse(msg)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case respondMsg:
		m.loading = false
		if msg.err != nil {
			// Handle AI request errors
			errorMsg := "❌ Error: " + msg.err.Error()
			m.messages = append(m.messages, message{
				sender: "ai",
				text:   errorMsg,
				mode:   m.viewMode,
			})
		} else {
			m.messages = append(m.messages, message{
				sender: "ai",
				text:   msg.response,
				mode:   m.viewMode,
			})
		}
		// Render messages with new content
		content := m.renderChatContent()
		m.viewport.SetContent(content)
		// Only scroll to bottom if user wasn't manually scrolling
		if !m.userScrolling {
			m.viewport.GotoBottom()
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if m.viewMode == ViewModeSelect {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height
		} else {
			// Resize viewport to fill remaining space after input
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 1
			m.input.Width = msg.Width

			// Re-render messages with new width
			if len(m.messages) > 0 {
				content := m.renderChatContent()
				m.viewport.SetContent(content)
				// Only scroll to bottom if user wasn't manually scrolling
				if !m.userScrolling {
					m.viewport.GotoBottom()
				}
			}
		}
		return m, nil
	}

	return m, nil
}

// updateModeSelect handles mode selection events
func (m model) updateModeSelect(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		return m, tea.Quit
	case tea.KeyUp:
		if m.selectedOption > 0 {
			m.selectedOption--
		}
	case tea.KeyDown:
		if m.selectedOption < len(modes)-1 {
			m.selectedOption++
		}
	case tea.KeyEnter:
		// Switch to selected mode
		m.viewMode = modes[m.selectedOption].Mode
		// Add welcome message
		selectedMode := modes[m.selectedOption]
		m.messages = append(m.messages, message{
			sender: "ai",
			text:   selectedMode.Welcome,
			mode:   m.viewMode,
		})
		// Update input placeholder
		m.input.Placeholder = selectedMode.Prompt
		// Setup viewport for chat
		m.viewport.Width = m.width
		m.viewport.Height = m.height - 1
		m.input.Width = m.width
		// Render welcome message
		content := m.renderChatContent()
		m.viewport.SetContent(content)
		m.viewport.GotoBottom()
		return m, nil
	}
	return m, nil
}

// updateModeSelectMouse handles mouse events in mode selection
func (m model) updateModeSelectMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.MouseWheelUp:
		// Scroll up - same as KeyUp
		if m.selectedOption > 0 {
			m.selectedOption--
		}
	case tea.MouseWheelDown:
		// Scroll down - same as KeyDown
		if m.selectedOption < len(modes)-1 {
			m.selectedOption++
		}
	}
	return m, nil
}

// updateChatMode handles chat mode events
func (m model) updateChatMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEsc:
		// Return to mode selection
		m.viewMode = ViewModeSelect
		m.viewport.Width = m.width
		m.viewport.Height = m.height
		return m, nil
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
			m.messages = append(m.messages, message{
				sender: "user",
				text:   userText,
				mode:   m.viewMode,
			})
			// Render and scroll viewport to bottom
			content := m.renderChatContent()
			m.viewport.SetContent(content)
			m.viewport.GotoBottom()
			m.userScrolling = false // Reset scrolling state when sending message
			m.input.Reset()
			m.loading = true
			return m, tea.Batch(
				spinner.Tick,
				makeAIRequest(userText, m.viewMode, m.agent, m.config.Model),
			)
		}
	}

	// Always update text input (allow typing even when loading)
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// updateChatModeMouse handles mouse events in chat mode
func (m model) updateChatModeMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.MouseWheelUp:
		// Scroll up in viewport - same as KeyUp
		m.viewport.LineUp(1)
		m.userScrolling = true
		return m, nil
	case tea.MouseWheelDown:
		// Scroll down in viewport - same as KeyDown
		m.viewport.LineDown(1)
		// Check if we scrolled back to the bottom
		if m.viewport.AtBottom() {
			m.userScrolling = false
		}
		return m, nil
	}
	return m, nil
}

// View renders the TUI.
func (m model) View() string {
	if m.viewMode == ViewModeSelect {
		return m.renderModeSelect()
	}
	return m.renderChatMode()
}

// renderModeSelect renders the mode selection interface
func (m model) renderModeSelect() string {
	var content string

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		Render("🦆 RubrDuck - Choose Your Mode")

	content += title + "\n\n"

	for i, mode := range modes {
		prefix := "  "
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

		if i == m.selectedOption {
			prefix = "❯ "
			style = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205"))
		}

		line := style.Render(prefix + mode.Icon + " " + mode.Name + " - " + mode.Description)
		content += line + "\n"
	}

	content += "\n"
	content += lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render("Use ↑/↓ to navigate, Enter to select, Ctrl+C to exit")

	return content
}

// renderChatMode renders the chat interface for the current mode
func (m model) renderChatMode() string {
	var inputView string
	if m.loading {
		// Show spinner with input field when AI is thinking
		spinner := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render(m.spinner.View() + " AI thinking... ")
		inputView = lipgloss.JoinHorizontal(lipgloss.Left, spinner, m.input.View())
	} else {
		inputView = m.input.View()
	}

	// Add mode indicator and back instruction
	currentMode := modes[m.selectedOption]
	if m.viewMode != ViewModeSelect {
		for _, mode := range modes {
			if mode.Mode == m.viewMode {
				currentMode = mode
				break
			}
		}
	}

	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Render(currentMode.Icon + " " + currentMode.Name + " Mode (ESC to return to mode selection)")

	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(inputView)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		m.viewport.View(),
		footer,
	)
}

// renderChatContent formats the chat history for the viewport
func (m model) renderChatContent() string {
	var out string

	// Filter messages for current mode
	var modeMessages []message
	for _, msg := range m.messages {
		if msg.mode == m.viewMode {
			modeMessages = append(modeMessages, msg)
		}
	}

	for _, msg := range modeMessages {
		var prefix string
		if msg.sender == "user" {
			prefix = lipgloss.NewStyle().
				Foreground(lipgloss.Color("2")).
				Bold(true).
				Render("You:   ")
		} else {
			prefix = lipgloss.NewStyle().
				Foreground(lipgloss.Color("4")).
				Bold(true).
				Render("AI:    ")
		}

		// Wrap text for better readability
		wrappedText := lipgloss.NewStyle().
			Width(m.viewport.Width - 7).
			Render(msg.text)

		out += prefix + wrappedText + "\n\n"
	}

	return out
}

// respondMsg carries an AI response.
type respondMsg struct {
	response string
	mode     ViewMode
	err      error
}

// makeAIRequest processes user input through the agent with tools
func makeAIRequest(input string, mode ViewMode, agent *agent.Agent, model string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		var response string
		var err error

		// Route to appropriate mode processor
		switch mode {
		case ViewModePlanning:
			response, err = ProcessPlanningRequest(ctx, agent, input, model)
		case ViewModeBuilding:
			response, err = ProcessBuildingRequest(ctx, agent, input, model)
		case ViewModeDebugging:
			response, err = ProcessDebuggingRequest(ctx, agent, input, model)
		case ViewModeEnhance:
			response, err = ProcessEnhanceRequest(ctx, agent, input, model)
		default:
			err = fmt.Errorf("unknown mode: %v", mode)
		}

		if err != nil {
			return respondMsg{
				response: "",
				mode:     mode,
				err:      err,
			}
		}

		return respondMsg{
			response: response,
			mode:     mode,
			err:      nil,
		}
	}
}
