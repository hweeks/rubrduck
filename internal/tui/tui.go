package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hammie/rubrduck/internal/agent"
	"github.com/hammie/rubrduck/internal/config"
)

// Enhanced Styles with better visual effects
var (
	// Title with gradient-like effect and better spacing
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			Background(lipgloss.Color("236")).
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("205")).
			Margin(0, 0, 1, 0)

	// Enhanced prompt with better visual hierarchy
	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Bold(true).
			Margin(0, 0, 0, 1)

	// Enhanced input with focus states and better borders
	inputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1).
			Margin(0, 0, 0, 1).
			Background(lipgloss.Color("235"))

	inputFocusedStyle = inputStyle.Copy().
				BorderForeground(lipgloss.Color("205")).
				Background(lipgloss.Color("236")).
				Foreground(lipgloss.Color("255")).
				Bold(true)

	// Enhanced message styles with better spacing and visual hierarchy
	messageStyle = lipgloss.NewStyle().
			PaddingLeft(1).
			PaddingRight(1).
			Margin(0, 0, 1, 0)

	userMessageStyle = messageStyle.Copy().
				Foreground(lipgloss.Color("170")).
				Border(lipgloss.RoundedBorder(), false, false, false, true).
				BorderForeground(lipgloss.Color("170")).
				Padding(0, 0, 0, 1).
				Background(lipgloss.Color("236")).
				Margin(0, 0, 1, 0)

	aiMessageStyle = messageStyle.Copy().
			Foreground(lipgloss.Color("86")).
			Border(lipgloss.RoundedBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("86")).
			Padding(0, 0, 0, 1).
			Background(lipgloss.Color("235")).
			Margin(0, 0, 1, 0)

	// Enhanced error styling
	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true).
			Background(lipgloss.Color("236")).
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("196"))

	// Enhanced help text with better readability
	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true).
			Margin(1, 0, 0, 0)

	// Enhanced status with better visibility
	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true).
			Background(lipgloss.Color("236")).
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))

	// Enhanced code block styling
	codeBlockStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1).
			Margin(0, 0, 1, 0)

	// Enhanced history item styling
	historyItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Padding(0, 1).
				Margin(0, 0, 0, 0)

	selectedHistoryItemStyle = historyItemStyle.Copy().
					Foreground(lipgloss.Color("205")).
					Background(lipgloss.Color("236")).
					Bold(true)

	// New styles for better visual effects
	dividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Margin(1, 0, 1, 0)

	highlightStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("205")).
			Foreground(lipgloss.Color("0")).
			Bold(true)

	// Animation and transition styles
	fadeInStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)

	// Mouse interaction styles
	clickableStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Underline(true)

	hoverStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Background(lipgloss.Color("236")).
			Bold(true)
)

type model struct {
	config   *config.Config
	agent    *agent.Agent
	messages []message
	input    textInput
	width    int
	height   int
	ready    bool
	quitting bool

	// Input history
	history      []string
	historyIndex int
	historyMode  bool

	// Multi-line input
	cursorPos   int
	lines       []string
	currentLine int

	// Status
	status      string
	statusTimer *time.Timer

	// View modes
	viewMode ViewMode

	// Search
	searchQuery   string
	searchMode    bool
	searchResults []int
	searchIndex   int

	// Enhanced features
	inputFocused bool
	mousePos     tea.MouseMsg
	hoveredItem  string
	animations   map[string]time.Time
	showCursor   bool
	cursorBlink  time.Time
}

type textInput struct {
	value   string
	cursor  int
	history []string
	index   int
}

type message struct {
	role      string // "user" or "assistant"
	content   string
	timestamp time.Time
	id        string
}

type ViewMode int

const (
	ViewModeChat ViewMode = iota
	ViewModeHistory
	ViewModeHelp
)

func initialModel(cfg *config.Config) model {
	// Create agent instance
	agentInstance, err := agent.New(cfg)
	if err != nil {
		// Handle error gracefully - we'll show it in the UI
		return model{
			config:   cfg,
			messages: []message{},
			input:    "",
		}
	}

	// Set up approval callback
	agentInstance.SetApprovalCallback(handleApproval)

		return model{
			config:       cfg,
			agent:        agentInstance,
			messages:     []message{},
			input:        textInput{},
			history:      []string{},
			lines:        []string{""},
			viewMode:     ViewModeChat,
			animations:   make(map[string]time.Time),
			showCursor:   true,
			inputFocused: true,
		}
}

// handleApproval handles approval requests from the agent
func handleApproval(req agent.ApprovalRequest) (agent.ApprovalResult, error) {
	// Show approval dialog
	result, err := ShowApprovalDialog(req)
	if err != nil {
		return agent.ApprovalResult{Approved: false, Reason: fmt.Sprintf("Dialog error: %v", err)}, err
	}

	return agent.ApprovalResult{
		Approved: result.Approved,
		Reason:   result.Reason,
	}, nil
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tea.SetWindowTitle("RubrDuck"),
		m.cursorBlinkCmd(),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.ready {
			m.ready = true
		}
	case tea.MouseMsg:
		return m.handleMouse(msg)
	case statusMsg:
		m.status = msg.message
		if m.statusTimer != nil {
			m.statusTimer.Stop()
		}
		m.statusTimer = time.AfterFunc(3*time.Second, func() {
			// Clear status after timeout
		})
	case cursorBlinkMsg:
		m.showCursor = !m.showCursor
		return m, m.cursorBlinkCmd()
	}

	return m, nil
}

func (m model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	m.mousePos = msg

	switch msg.Type {
	case tea.MouseMotion:
		// Handle hover effects
		m.hoveredItem = m.getItemAtPosition(msg.X, msg.Y)
	case tea.MouseRelease:
		// Handle clicks
		if item := m.getItemAtPosition(msg.X, msg.Y); item != "" {
			return m.handleItemClick(item)
		}
	}

	return m, nil
}

func (m model) getItemAtPosition(x, y int) string {
	// Simple position-based item detection
	// This could be enhanced with more sophisticated layout tracking
	if y > m.height-5 && x < 20 {
		return "help"
	}
	return ""
}

func (m model) handleItemClick(item string) (tea.Model, tea.Cmd) {
	switch item {
	case "help":
		m.viewMode = ViewModeHelp
		return m, m.setStatus("Help view opened")
	}
	return m, nil
}

func (m model) cursorBlinkCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return cursorBlinkMsg(t)
	})
}

type cursorBlinkMsg time.Time

func (m model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.viewMode {
	case ViewModeChat:
		return m.handleChatKeyPress(msg)
	case ViewModeHistory:
		return m.handleHistoryKeyPress(msg)
	case ViewModeHelp:
		return m.handleHelpKeyPress(msg)
	}
	return m, nil
}

func (m model) handleChatKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		if m.searchMode {
			m.searchMode = false
			m.searchQuery = ""
			m.searchResults = nil
			m.inputFocused = true
			return m, m.setStatus("Search cancelled")
		}
		m.quitting = true
		return m, tea.Quit

	case tea.KeyEnter:
		// For now, just submit message (Shift+Enter will be handled differently)
		if m.input.value != "" {
			m.inputFocused = false
			return m.submitMessage()
		}

	case tea.KeyUp:
		// Regular up arrow for line navigation
		return m.navigateLines(-1)

	case tea.KeyDown:
		// Regular down arrow for line navigation
		return m.navigateLines(1)

	case tea.KeyCtrlUp:
		// Ctrl+Up for history navigation
		return m.navigateHistory(-1)

	case tea.KeyCtrlDown:
		// Ctrl+Down for history navigation
		return m.navigateHistory(1)

	case tea.KeyCtrlR:
		// Search history
		m.searchMode = true
		m.searchQuery = ""
		m.searchResults = m.searchHistory("")
		m.inputFocused = false
		return m, m.setStatus("Search mode activated")

	case tea.KeyCtrlL:
		// Clear screen
		return m, tea.ClearScreen

	case tea.KeyCtrlH:
		// Show history view
		m.viewMode = ViewModeHistory
		m.inputFocused = false
		return m, m.setStatus("History view opened")

	case tea.KeyF1:
		// Show help
		m.viewMode = ViewModeHelp
		m.inputFocused = false
		return m, m.setStatus("Help view opened")

	case tea.KeyTab:
		// Toggle focus
		m.inputFocused = !m.inputFocused
		return m, m.setStatus("Focus toggled")

	case tea.KeyBackspace:
		if len(m.input.value) > 0 {
			m.input.value = m.input.value[:len(m.input.value)-1]
		}

	default:
		if m.searchMode {
			return m.handleSearchInput(msg)
		}
		if m.inputFocused {
			m.input.value += msg.String()
		}
	}

	return m, nil
}

func (m model) handleSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		if len(m.searchResults) > 0 {
			selectedIndex := m.searchResults[m.searchIndex]
			if selectedIndex < len(m.history) {
				m.input.value = m.history[selectedIndex]
				m.inputFocused = true
			}
		}
		m.searchMode = false
		m.searchQuery = ""
		m.searchResults = nil
		return m, m.setStatus("Search completed")

	case tea.KeyEsc:
		m.searchMode = false
		m.searchQuery = ""
		m.searchResults = nil
		m.inputFocused = true
		return m, m.setStatus("Search cancelled")

	case tea.KeyUp:
		if m.searchIndex > 0 {
			m.searchIndex--
		}
		return m, nil

	case tea.KeyDown:
		if m.searchIndex < len(m.searchResults)-1 {
			m.searchIndex++
		}
		return m, nil

	case tea.KeyBackspace:
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
			m.searchResults = m.searchHistory(m.searchQuery)
			m.searchIndex = 0
		}
		return m, nil

	default:
		m.searchQuery += msg.String()
		m.searchResults = m.searchHistory(m.searchQuery)
		m.searchIndex = 0
		return m, nil
	}
}

func (m model) handleHistoryKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlC:
		m.viewMode = ViewModeChat
		m.inputFocused = true
		return m, m.setStatus("Returned to chat")
	}
	return m, nil
}

func (m model) handleHelpKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlC:
		m.viewMode = ViewModeChat
		m.inputFocused = true
		return m, m.setStatus("Returned to chat")
	}
	return m, nil
}

func (m model) submitMessage() (tea.Model, tea.Cmd) {
	// Add to history
	if m.input.value != "" {
		m.history = append(m.history, m.input.value)
		if len(m.history) > 100 { // Keep last 100 messages
			m.history = m.history[1:]
		}
	}

	// Add user message
	userMsg := message{
		role:      "user",
		content:   m.input.value,
		timestamp: time.Now(),
		id:        fmt.Sprintf("user_%d", len(m.messages)),
	}
	m.messages = append(m.messages, userMsg)

	// Clear input
	m.input.value = ""
	m.input.cursor = 0

	// TODO: Send to AI and get response
	// For now, just echo back with enhanced formatting
	aiResponse := m.generateMockResponse(userMsg.content)
	aiMsg := message{
		role:      "assistant",
		content:   aiResponse,
		timestamp: time.Now(),
		id:        fmt.Sprintf("ai_%d", len(m.messages)),
	}
	m.messages = append(m.messages, aiMsg)

	// Restore input focus after sending message
	m.inputFocused = true

	return m, m.setStatus("Message sent")
}

func (m model) generateMockResponse(userInput string) string {
	// Enhanced mock response with different content types
	if strings.Contains(strings.ToLower(userInput), "hello") {
		return "Hello! I'm RubrDuck, your AI coding assistant. How can I help you today?\n\nI can help with:\n- Code generation and review\n- Bug fixing and debugging\n- Project structure analysis\n- Documentation writing"
	}

	if strings.Contains(strings.ToLower(userInput), "code") {
		return "Here's an example of how I can help with code:\n\n```go\nfunc main() {\n    fmt.Println(\"Hello, World!\")\n}\n```\n\nI can generate, review, and explain code in many programming languages."
	}

	if strings.Contains(strings.ToLower(userInput), "help") {
		return "## Available Commands\n\n- **Ctrl+Up/Down**: Navigate input history\n- **Ctrl+R**: Search history\n- **Ctrl+H**: View conversation history\n- **F1**: Show help\n- **Shift+Enter**: New line in input\n- **Ctrl+L**: Clear screen\n\n## Features\n\n- Multi-line input support\n- Syntax highlighting\n- Markdown rendering\n- Conversation management\n- History search"
	}

	return fmt.Sprintf("I received: %s\n\nThis is a mock response. AI integration is coming soon!\n\n**Features available:**\n- ✅ Multi-line input\n- ✅ History navigation\n- ✅ Rich formatting\n- ✅ Search functionality\n- 🔄 AI integration (in progress)", userInput)
}

func (m model) navigateHistory(direction int) (tea.Model, tea.Cmd) {
	if len(m.history) == 0 {
		return m, nil
	}

	m.input.index += direction
	if m.input.index < 0 {
		m.input.index = len(m.history) - 1
	} else if m.input.index >= len(m.history) {
		m.input.index = 0
	}

	if m.input.index >= 0 && m.input.index < len(m.history) {
		m.input.value = m.history[m.input.index]
		m.input.cursor = len(m.input.value)
	}

	return m, nil
}

func (m model) navigateLines(direction int) (tea.Model, tea.Cmd) {
	// For now, just move cursor within current line
	// TODO: Implement multi-line navigation
	return m, nil
}

func (m *model) addNewLineAbove() {
	if m.currentLine > 0 {
		newLines := make([]string, len(m.lines)+1)
		copy(newLines, m.lines[:m.currentLine])
		newLines[m.currentLine] = ""
		copy(newLines[m.currentLine+1:], m.lines[m.currentLine:])
		m.lines = newLines
		m.currentLine = m.currentLine // stays at the new blank line
	} else {
		m.lines = append([]string{""}, m.lines...)
		m.currentLine = 0
	}
}

func (m *model) addNewLineBelow() {
	if m.currentLine < len(m.lines)-1 {
		newLines := make([]string, len(m.lines)+1)
		copy(newLines, m.lines[:m.currentLine+1])
		newLines[m.currentLine+1] = ""
		copy(newLines[m.currentLine+2:], m.lines[m.currentLine+1:])
		m.lines = newLines
		m.currentLine++
	} else {
		m.lines = append(m.lines, "")
		m.currentLine = len(m.lines) - 1
	}
}

func (m model) searchHistory(query string) []int {
	if query == "" {
		return []int{}
	}

	var results []int
	query = strings.ToLower(query)

	for i, item := range m.history {
		if strings.Contains(strings.ToLower(item), query) {
			results = append(results, i)
		}
	}

	return results
}

func (m model) setStatus(message string) tea.Cmd {
	return func() tea.Msg {
		return statusMsg{message: message}
	}
}

type statusMsg struct {
	message string
}

func (m model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	if !m.ready {
		return "Initializing..."
	}

	switch m.viewMode {
	case ViewModeChat:
		return m.renderChatView()
	case ViewModeHistory:
		return m.renderHistoryView()
	case ViewModeHelp:
		return m.renderHelpView()
	}

	return m.renderChatView()
}

func (m model) renderChatView() string {
	var s strings.Builder

	// Enhanced title with better spacing
	title := titleStyle.Render("🦆 RubrDuck - AI Coding Assistant")
	s.WriteString(title + "\n")

	// Add divider
	s.WriteString(dividerStyle.Render(strings.Repeat("─", m.width)) + "\n")

	// Messages with better layout
	messageArea := m.renderMessages()
	s.WriteString(messageArea)

	// Add divider before input
	s.WriteString(dividerStyle.Render(strings.Repeat("─", m.width)) + "\n")

	// Enhanced input area
	inputArea := m.renderInputArea()
	s.WriteString(inputArea)

	// Enhanced status and help area
	statusArea := m.renderStatusArea()
	s.WriteString(statusArea)

	return s.String()
}

func (m model) renderMessages() string {
	var s strings.Builder

	// Calculate available space for messages
	availableHeight := m.height - 16 // Reserve more space for enhanced layout with better input area

	// Show last N messages that fit
	startIndex := 0
	if len(m.messages) > availableHeight/4 { // More space per message for better formatting
		startIndex = len(m.messages) - availableHeight/4
	}

	for i := startIndex; i < len(m.messages); i++ {
		msg := m.messages[i]
		s.WriteString(m.renderMessage(msg))
		s.WriteString("\n")
	}

	return s.String()
}

func (m model) renderMessage(msg message) string {
	var style lipgloss.Style
	var prefix string
	var icon string

	if msg.role == "user" {
		style = userMessageStyle
		prefix = "You"
		icon = "👤"
	} else {
		style = aiMessageStyle
		prefix = "RubrDuck"
		icon = "🤖"
	}

	// Format timestamp with better styling
	timestamp := msg.timestamp.Format("15:04")

	// Format content with enhanced markdown and code highlighting
	content := m.formatContent(msg.content)

	// Create enhanced header with icon and timestamp
	header := fmt.Sprintf("%s %s • %s", icon, prefix, timestamp)

	// Combine header and content with better spacing
	fullContent := fmt.Sprintf("%s\n%s", header, content)

	return style.Render(fullContent)
}

func (m model) formatContent(content string) string {
	// Enhanced markdown and code formatting
	lines := strings.Split(content, "\n")
	var formattedLines []string

	for _, line := range lines {
		// Code blocks with enhanced styling
		if strings.HasPrefix(line, "```") {
			formattedLines = append(formattedLines, codeBlockStyle.Render(line))
		} else if strings.Contains(line, "`") {
			// Enhanced inline code with better contrast
			parts := strings.Split(line, "`")
			for i, part := range parts {
				if i%2 == 1 { // Odd indices are code
					parts[i] = lipgloss.NewStyle().
						Background(lipgloss.Color("236")).
						Foreground(lipgloss.Color("255")).
						Padding(0, 1).
						Render(part)
				}
			}
			formattedLines = append(formattedLines, strings.Join(parts, ""))
		} else if strings.HasPrefix(line, "##") {
			// Enhanced H2 headers
			formattedLines = append(formattedLines, lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205")).
				Background(lipgloss.Color("236")).
				Padding(0, 1).
				Render(line))
		} else if strings.HasPrefix(line, "#") {
			// Enhanced H1 headers
			formattedLines = append(formattedLines, lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205")).
				Background(lipgloss.Color("236")).
				Padding(0, 1).
				Render(line))
		} else if strings.HasPrefix(line, "**") && strings.HasSuffix(line, "**") {
			// Enhanced bold text
			formattedLines = append(formattedLines, lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("255")).
				Render(line))
		} else if strings.HasPrefix(line, "- ") {
			// Enhanced list items with better bullets
			formattedLines = append(formattedLines, lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Render("• ")+line[2:])
		} else if strings.Contains(line, "✅") || strings.Contains(line, "🔄") || strings.Contains(line, "⚠️") || strings.Contains(line, "❌") {
			// Enhanced status indicators
			formattedLines = append(formattedLines, lipgloss.NewStyle().
				Bold(true).
				Render(line))
		} else {
			formattedLines = append(formattedLines, line)
		}
	}

	return strings.Join(formattedLines, "\n")
}

func (m model) renderInputArea() string {
	var s strings.Builder

	// Enhanced search mode indicator
	if m.searchMode {
		s.WriteString(promptStyle.Render("🔍 Search: "))
		input := inputFocusedStyle.Render(m.searchQuery)
		if m.showCursor {
			input += "█"
		}
		s.WriteString(input)
		if len(m.searchResults) > 0 {
			s.WriteString(fmt.Sprintf(" (%d results)", len(m.searchResults)))
		}
		s.WriteString("\n")

		// Enhanced search results with better styling
		for i, resultIndex := range m.searchResults {
			if i >= 5 { // Show max 5 results
				break
			}
			if resultIndex < 0 || resultIndex >= len(m.history) {
				continue
			}
			style := historyItemStyle
			if i == m.searchIndex {
				style = selectedHistoryItemStyle
			}
			preview := m.history[resultIndex]
			if len(preview) > 50 {
				preview = preview[:47] + "..."
			}
			s.WriteString(style.Render(fmt.Sprintf("%d: %s", i+1, preview)) + "\n")
		}
	} else {
		// Enhanced normal input with proper text wrapping
		s.WriteString("\n") // Add space before input
		prompt := promptStyle.Render("> ")

		// Calculate available width for input (accounting for prompt and borders)
		availableWidth := m.width - 10 // Reserve space for prompt, borders, and padding
		if availableWidth < 20 {
			availableWidth = 20 // Minimum width
		}

		// Wrap the input text properly
		wrappedLines := m.wrapText(m.input.value, availableWidth)

		style := inputStyle
		if m.inputFocused {
			style = inputFocusedStyle
		}

		// Render each line of the wrapped input
		for i, line := range wrappedLines {
			if i == 0 {
				// First line with prompt
				input := style.Render(line)
				if m.showCursor && m.inputFocused {
					// Add cursor at the appropriate position
					cursorPos := m.input.cursor
					if cursorPos > len(line) {
						cursorPos = len(line)
					}
					input = input[:cursorPos] + "█" + input[cursorPos:]
				}
				s.WriteString(prompt + input + "\n")
			} else {
				// Subsequent lines with proper indentation
				indent := strings.Repeat(" ", len(prompt))
				input := style.Render(line)
				s.WriteString(indent + input + "\n")
			}
		}

		// If no lines were rendered, show at least one empty line
		if len(wrappedLines) == 0 {
			input := style.Render("")
			if m.showCursor && m.inputFocused {
				input += "█"
			}
			s.WriteString(prompt + input + "\n")
		}

		s.WriteString("\n") // Add space after input
	}

	return s.String()
}

// wrapText wraps text to fit within the specified width
func (m model) wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	var lines []string
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	currentLine := words[0]

	for _, word := range words[1:] {
		// Check if adding this word would exceed the width
		if len(currentLine)+1+len(word) <= width {
			currentLine += " " + word
		} else {
			// Start a new line
			lines = append(lines, currentLine)
			currentLine = word
		}
	}

	// Add the last line
	lines = append(lines, currentLine)

	return lines
}

func (m model) renderStatusArea() string {
	var s strings.Builder

	// Enhanced status line with better visibility
	if m.status != "" {
		s.WriteString(statusStyle.Render("Status: "+m.status) + "\n")
	}

	// Enhanced help text with clickable elements
	helpText := "Ctrl+H: History | Ctrl+R: Search | F1: Help | Ctrl+C: Quit"
	if m.hoveredItem == "help" {
		helpText = hoverStyle.Render(helpText)
	} else {
		helpText = helpStyle.Render(helpText)
	}
	s.WriteString(helpText)

	return s.String()
}

func (m model) renderHistoryView() string {
	var s strings.Builder

	// Enhanced title
	title := titleStyle.Render("📚 Conversation History")
	s.WriteString(title + "\n")

	// Add divider
	s.WriteString(dividerStyle.Render(strings.Repeat("─", m.width)) + "\n")

	if len(m.history) == 0 {
		s.WriteString(fadeInStyle.Render("No conversation history yet.\n"))
	} else {
		for i, item := range m.history {
			preview := item
			if len(preview) > 80 {
				preview = preview[:77] + "..."
			}
			s.WriteString(historyItemStyle.Render(fmt.Sprintf("%d: %s", i+1, preview)) + "\n")
		}
	}

	s.WriteString("\n" + helpStyle.Render("Press ESC to return to chat"))

	return s.String()
}

func (m model) renderHelpView() string {
	var s strings.Builder

	// Enhanced title
	title := titleStyle.Render("❓ Help & Keyboard Shortcuts")
	s.WriteString(title + "\n")

	// Add divider
	s.WriteString(dividerStyle.Render(strings.Repeat("─", m.width)) + "\n")

	helpContent := `## Navigation
- **Ctrl+Up/Down**: Navigate input history
- **Ctrl+R**: Search through conversation history
- **Ctrl+H**: View conversation history
- **F1**: Show this help

## Input
- **Shift+Enter**: Add new line in input
- **Ctrl+L**: Clear screen
- **Ctrl+C**: Quit application

## Features
- **Multi-line input**: Use Shift+Enter for new lines
- **History search**: Press Ctrl+R and type to search
- **Rich formatting**: Supports basic markdown and code highlighting
- **Conversation management**: Save and load conversations (coming soon)
- **Mouse support**: Click on elements for quick access

## AI Integration
- **Code generation**: Ask for code examples
- **Code review**: Submit code for review
- **Bug fixing**: Describe issues for solutions
- **Documentation**: Request documentation help

## Status Indicators
- 🟢 Ready for input
- 🔄 Processing request
- ⚠️  Warning or notice
- ❌ Error occurred`

	s.WriteString(m.formatContent(helpContent))
	s.WriteString("\n\n" + helpStyle.Render("Press ESC to return to chat"))

	return s.String()
}

// Run starts the TUI application
func Run(cfg *config.Config) error {
	p := tea.NewProgram(
		initialModel(cfg),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(), // Enable mouse support
	)
	_, err := p.Run()
	return err
}
