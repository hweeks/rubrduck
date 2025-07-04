package tui2

import (
	"context"
	"fmt"
	"strings"
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
	// Create a program reference that we can use for sending messages
	var program *tea.Program

	// Initialize AI agent with tools
	ag, err := agent.New(cfg)
	if err != nil {
		return err
	}

	// Create approval callback that integrates with the TUI
	approvalCallback := func(req agent.ApprovalRequest) (agent.ApprovalResult, error) {
		// Create a response channel
		responseChan := make(chan approvalResponse, 1)

		// Send approval request to the UI
		program.Send(approvalRequestMsg{
			request:  req,
			response: responseChan,
		})

		// Wait for user response with timeout
		select {
		case resp := <-responseChan:
			return resp.result, resp.err
		case <-time.After(5 * time.Minute):
			return agent.ApprovalResult{
				Approved: false,
				Reason:   "Approval request timed out",
			}, nil
		}
	}

	// Set the approval callback on the agent
	ag.SetApprovalCallback(approvalCallback)

	// Create the program with the model
	program = tea.NewProgram(
		newModel(cfg, ag),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	_, err = program.Run()
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
	streaming      bool
	partial        string
	streamCh       <-chan agent.StreamEvent
	streamChunks   int // Track number of chunks received

	// Approval state
	showingApproval bool                   // Whether we're showing approval dialog
	approvalRequest *agent.ApprovalRequest // Current approval request
	approvalChan    chan approvalResponse  // Channel to send approval response

	// Dimensions
	width  int
	height int

	// AI integration
	config *config.Config
	agent  *agent.Agent
}

// approvalResponse carries the user's approval decision
type approvalResponse struct {
	result agent.ApprovalResult
	err    error
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
		spinner:         s,
		viewport:        vp,
		input:           ti,
		viewMode:        ViewModeSelect,
		selectedOption:  0,
		messages:        make([]message, 0),
		loading:         false,
		userScrolling:   false,
		streaming:       false,
		partial:         "",
		streamCh:        nil,
		streamChunks:    0,
		showingApproval: false,
		approvalRequest: nil,
		approvalChan:    nil,
		config:          cfg,
		agent:           agent,
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
		if m.showingApproval {
			return m.updateApprovalMode(msg)
		}
		if m.viewMode == ViewModeSelect {
			return m.updateModeSelect(msg)
		}
		return m.updateChatMode(msg)

	case tea.MouseMsg:
		if m.showingApproval {
			// Ignore mouse events during approval
			return m, nil
		}
		if m.viewMode == ViewModeSelect {
			return m.updateModeSelectMouse(msg)
		}
		return m.updateChatModeMouse(msg)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case approvalRequestMsg:
		// Show approval dialog
		m.showingApproval = true
		m.approvalRequest = &msg.request
		m.approvalChan = msg.response
		// Clear input for approval response
		m.input.Reset()
		m.input.Placeholder = "y=approve, n=deny, e=explain denial..."
		m.input.Focus()
		return m, nil

	case streamMsg:
		switch msg.event.Type {
		case agent.EventTokenChunk:
			m.streamChunks++
			m.partial += msg.event.Token

			// Show progress for large operations
			progressIndicator := ""
			if m.streamChunks > 100 {
				progressIndicator = fmt.Sprintf(" [%d chunks received]", m.streamChunks)
			}

			content := m.renderChatContent() + lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Bold(true).Render("AI:    ") + m.partial + progressIndicator
			m.viewport.SetContent(content)
			if !m.userScrolling {
				m.viewport.GotoBottom()
			}
			if msg.cancel != nil {
				return m, listenStreamWithCancel(msg.ch, msg.cancel)
			}
			return m, listenStream(msg.ch)
		case agent.EventToolResult:
			m.streamChunks++
			m.partial += "\n" + msg.event.Result

			// Show progress for large operations
			progressIndicator := ""
			if m.streamChunks > 100 {
				progressIndicator = fmt.Sprintf(" [%d chunks received]", m.streamChunks)
			}

			content := m.renderChatContent() + lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Bold(true).Render("AI:    ") + m.partial + progressIndicator
			m.viewport.SetContent(content)
			if !m.userScrolling {
				m.viewport.GotoBottom()
			}
			if msg.cancel != nil {
				return m, listenStreamWithCancel(msg.ch, msg.cancel)
			}
			return m, listenStream(msg.ch)
		case agent.EventDone:
			m.loading = false
			m.streaming = false
			m.messages = append(m.messages, message{sender: "ai", text: m.partial, mode: m.viewMode})
			m.partial = ""
			m.streamChunks = 0 // Reset chunk counter
			content := m.renderChatContent()
			m.viewport.SetContent(content)
			if !m.userScrolling {
				m.viewport.GotoBottom()
			}
			// Context should already be canceled by listenStreamWithCancel
			return m, nil
		default:
			if msg.cancel != nil {
				return m, listenStreamWithCancel(msg.ch, msg.cancel)
			}
			return m, listenStream(msg.ch)
		}

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

// updateApprovalMode handles user input during approval requests
func (m model) updateApprovalMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		// Cancel the approval request
		if m.approvalChan != nil {
			m.approvalChan <- approvalResponse{
				result: agent.ApprovalResult{
					Approved: false,
					Reason:   "User cancelled approval",
				},
				err: nil,
			}
		}
		m.showingApproval = false
		m.approvalRequest = nil
		m.approvalChan = nil
		m.input.Reset()
		m.input.Placeholder = modes[m.selectedOption].Prompt
		return m, nil

	case tea.KeyEnter:
		response := strings.ToLower(strings.TrimSpace(m.input.Value()))

		switch {
		case response == "y" || response == "yes":
			// Approve
			if m.approvalChan != nil {
				m.approvalChan <- approvalResponse{
					result: agent.ApprovalResult{
						Approved: true,
						Reason:   "User approved",
					},
					err: nil,
				}
			}
			m.showingApproval = false
			m.approvalRequest = nil
			m.approvalChan = nil
			m.input.Reset()
			m.input.Placeholder = modes[m.selectedOption].Prompt
			return m, nil

		case response == "n" || response == "no":
			// Deny
			if m.approvalChan != nil {
				m.approvalChan <- approvalResponse{
					result: agent.ApprovalResult{
						Approved: false,
						Reason:   "User denied",
					},
					err: nil,
				}
			}
			m.showingApproval = false
			m.approvalRequest = nil
			m.approvalChan = nil
			m.input.Reset()
			m.input.Placeholder = modes[m.selectedOption].Prompt
			return m, nil

		case strings.HasPrefix(response, "e "):
			// Deny with explanation
			reason := strings.TrimPrefix(response, "e ")
			if reason == "" {
				reason = "User denied with no explanation"
			}
			if m.approvalChan != nil {
				m.approvalChan <- approvalResponse{
					result: agent.ApprovalResult{
						Approved: false,
						Reason:   reason,
					},
					err: nil,
				}
			}
			m.showingApproval = false
			m.approvalRequest = nil
			m.approvalChan = nil
			m.input.Reset()
			m.input.Placeholder = modes[m.selectedOption].Prompt
			return m, nil

		default:
			// Clear input if invalid response
			if response != "" && response != "e" {
				m.input.Reset()
			}
		}
	}

	// Update text input
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
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
			m.streaming = true
			m.streamChunks = 0 // Reset chunk counter for new stream
			cmd := makeAIRequest(userText, m.viewMode, m.agent, m.config)
			return m, tea.Batch(spinner.Tick, cmd)
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
	if m.showingApproval {
		return m.renderApprovalDialog()
	}
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
		spinnerText := " AI thinking... "
		if m.streamChunks > 0 {
			spinnerText = fmt.Sprintf(" AI processing... [%d chunks] ", m.streamChunks)
		}
		spinner := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render(m.spinner.View() + spinnerText)
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

	// Get timeout for current mode
	var timeout int
	switch m.viewMode {
	case ViewModePlanning:
		timeout = m.config.TUI.PlanningTimeout
	case ViewModeBuilding:
		timeout = m.config.TUI.BuildingTimeout
	case ViewModeDebugging:
		timeout = m.config.TUI.DebugTimeout
	case ViewModeEnhance:
		timeout = m.config.TUI.EnhanceTimeout
	}

	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Render(fmt.Sprintf("%s %s Mode (timeout: %ds) - ESC to return", currentMode.Icon, currentMode.Name, timeout))

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

// renderApprovalDialog renders the approval request interface
func (m model) renderApprovalDialog() string {
	if m.approvalRequest == nil {
		return "Error: No approval request"
	}

	req := m.approvalRequest

	// Title with risk indicator
	var riskColor string
	switch req.Risk {
	case agent.RiskLow:
		riskColor = "2" // Green
	case agent.RiskMedium:
		riskColor = "3" // Yellow
	case agent.RiskHigh:
		riskColor = "202" // Orange
	case agent.RiskCritical:
		riskColor = "1" // Red
	default:
		riskColor = "7" // White
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		Render("⚠️  Approval Required")

	riskBadge := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(riskColor)).
		Background(lipgloss.Color("235")).
		Padding(0, 1).
		Render(fmt.Sprintf("%s RISK", strings.ToUpper(string(req.Risk))))

	// Operation details
	operationType := lipgloss.NewStyle().
		Bold(true).
		Render("Operation: ") + req.Type

	description := lipgloss.NewStyle().
		Bold(true).
		Render("Description: ") + req.Description

	// Tool and preview
	tool := lipgloss.NewStyle().
		Bold(true).
		Render("Tool: ") + req.Tool

	preview := lipgloss.NewStyle().
		Bold(true).
		Render("Preview:\n") +
		lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			MarginLeft(2).
			Render(req.Preview)

	// Build the dialog content
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title+" "+riskBadge,
		"",
		operationType,
		description,
		tool,
		"",
		preview,
		"",
		lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Render("─────────────────────────────────────────"),
		"",
		lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("2")).
			Render("Response Options:"),
		lipgloss.NewStyle().
			MarginLeft(2).
			Render("y/yes  - Approve this operation\n"+
				"n/no   - Deny this operation\n"+
				"e <reason> - Deny with explanation"),
		"",
		m.input.View(),
	)

	// Center the dialog
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(riskColor)).
		Padding(1, 2).
		Width(m.width - 4).
		MaxWidth(100)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		dialogStyle.Render(content),
	)
}

// respondMsg carries an AI response.
type respondMsg struct {
	response string
	mode     ViewMode
	err      error
}

// approvalRequestMsg carries an approval request from the agent
type approvalRequestMsg struct {
	request  agent.ApprovalRequest
	response chan approvalResponse
}

type streamMsg struct {
	event  agent.StreamEvent
	ch     <-chan agent.StreamEvent
	cancel context.CancelFunc
}

// makeAIRequest processes user input through the agent with tools
func makeAIRequest(input string, mode ViewMode, ag *agent.Agent, cfg *config.Config) tea.Cmd {
	return func() tea.Msg {
		// Get timeout from config based on mode
		var timeout time.Duration

		switch mode {
		case ViewModePlanning:
			if cfg.TUI.PlanningTimeout > 0 {
				timeout = time.Duration(cfg.TUI.PlanningTimeout) * time.Second
			} else {
				timeout = time.Duration(cfg.Agent.Timeout) * time.Second
			}
		case ViewModeBuilding:
			if cfg.TUI.BuildingTimeout > 0 {
				timeout = time.Duration(cfg.TUI.BuildingTimeout) * time.Second
			} else {
				timeout = time.Duration(cfg.Agent.Timeout) * time.Second
			}
		case ViewModeDebugging:
			if cfg.TUI.DebugTimeout > 0 {
				timeout = time.Duration(cfg.TUI.DebugTimeout) * time.Second
			} else {
				timeout = time.Duration(cfg.Agent.Timeout) * time.Second
			}
		case ViewModeEnhance:
			if cfg.TUI.EnhanceTimeout > 0 {
				timeout = time.Duration(cfg.TUI.EnhanceTimeout) * time.Second
			} else {
				timeout = time.Duration(cfg.Agent.Timeout) * time.Second
			}
		default:
			timeout = time.Duration(cfg.Agent.Timeout) * time.Second
		}

		// Create a context with appropriate timeout
		ctx, cancel := context.WithTimeout(context.Background(), timeout)

		var ch <-chan agent.StreamEvent
		var err error

		switch mode {
		case ViewModePlanning:
			ch, err = ProcessPlanningRequest(ctx, ag, input, cfg.Model)
		case ViewModeBuilding:
			ch, err = ProcessBuildingRequest(ctx, ag, input, cfg.Model)
		case ViewModeDebugging:
			ch, err = ProcessDebuggingRequest(ctx, ag, input, cfg.Model)
		case ViewModeEnhance:
			ch, err = ProcessEnhanceRequest(ctx, ag, input, cfg.Model)
		default:
			err = fmt.Errorf("unknown mode: %v", mode)
		}

		if err != nil {
			cancel() // Cancel on error
			return respondMsg{response: "", mode: mode, err: err}
		}

		ev, ok := <-ch
		if !ok {
			cancel() // Cancel when stream ends
			return respondMsg{response: "", mode: mode, err: nil}
		}
		return streamMsg{event: ev, ch: ch, cancel: cancel}
	}
}

func listenStream(ch <-chan agent.StreamEvent) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return streamMsg{event: agent.StreamEvent{Type: agent.EventDone}, ch: nil}
		}
		return streamMsg{event: ev, ch: ch}
	}
}

func listenStreamWithCancel(ch <-chan agent.StreamEvent, cancel context.CancelFunc) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			if cancel != nil {
				cancel() // Cancel context when stream ends
			}
			return streamMsg{event: agent.StreamEvent{Type: agent.EventDone}, ch: nil}
		}
		return streamMsg{event: ev, ch: ch, cancel: cancel}
	}
}
