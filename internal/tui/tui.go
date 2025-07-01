package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
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

	// huh form for input
	form        *huh.Form
	inputValue  string
	showingForm bool

	quitting     bool
	responses    []Response // History of AI responses
	scrollOffset int        // For scrolling through responses
}

// NewModel initializes a new TUI model using provided configuration
func NewModel(cfg *config.Config) Model {
	m := Model{
		config:         cfg,
		selectedOption: 0,
		responses:      make([]Response, 0),
		scrollOffset:   0,
		showingForm:    true,
		inputValue:     "",
	}

	// Create the input form that will stay at the bottom
	m.createInputForm()

	// Determine initial view mode from config or prompt selection
	if s := strings.ToLower(cfg.TUI.StartMode); s != "" {
		switch s {
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
	} else {
		m.viewMode = ViewModeSelect
	}
	return m
}

// createInputForm sets up the huh form for user input
func (m *Model) createInputForm() {
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Key("query").
				Title("💬 Your message").
				Placeholder("Type your question or command...").
				Value(&m.inputValue).
				Validate(func(str string) error {
					if strings.TrimSpace(str) == "" {
						return fmt.Errorf("message cannot be empty")
					}
					return nil
				}),
		),
	).WithTheme(huh.ThemeCharm())
}

// Init is called when the TUI starts; it sets up the alternate screen
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		tea.ClearScreen,
		m.form.Init(),
	)
}

// Update processes messages and delegates to the active mode handler
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmdForm tea.Cmd
	var cmdMode tea.Cmd

	// Handle global key events first
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "esc":
			// Handle escape in mode-specific ways
			if m.viewMode != ViewModeSelect {
				m.viewMode = ViewModeSelect
				return m, nil
			}
		}
	case tea.MouseMsg:
		// Scroll with mouse wheel in the content area (not the form)
		if msg.Y < m.height-8 { // Above the form area
			if msg.Button == tea.MouseButtonWheelUp {
				if m.scrollOffset > 0 {
					m.scrollOffset--
				}
			} else if msg.Button == tea.MouseButtonWheelDown {
				m.scrollOffset++
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	// Update the form
	if m.showingForm {
		form, cmd := m.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.form = f
		}
		cmdForm = cmd

		// Check if form is completed (user pressed enter)
		if m.form.State == huh.StateCompleted {
			// Process the input
			query := strings.TrimSpace(m.inputValue)
			if query != "" {
				// Add user message
				m.responses = append(m.responses, Response{
					Mode:  m.viewMode,
					Query: query,
					Time:  "Just now",
				})

				// Generate AI response
				assistant := m.generateSimulatedResponse(query)
				m.responses = append(m.responses, assistant)

				// Reset the form for next input
				m.inputValue = ""
				m.createInputForm()
				cmdForm = tea.Batch(cmdForm, m.form.Init())

				// Scroll to show the new response
				m.adjustScrollToBottom()
			}
		}
	}

	// Keep scrollOffset within valid bounds
	m.clampScroll()

	// Delegate to mode-specific handlers
	switch m.viewMode {
	case ViewModeSelect:
		m, cmdMode = m.updateModeSelect(msg)
	case ViewModePlanning:
		m, cmdMode = m.updatePlanning(msg)
	case ViewModeBuilding:
		m, cmdMode = m.updateBuilding(msg)
	case ViewModeDebugging:
		m, cmdMode = m.updateDebugging(msg)
	case ViewModeTechDebt:
		m, cmdMode = m.updateTechDebt(msg)
	}

	return m, tea.Batch(cmdForm, cmdMode)
}

// View renders the active mode screen along with input and help bars
func (m Model) View() string {
	if m.quitting {
		return "Thanks for using RubrDuck! 🦆\n"
	}

	// Calculate available height for content (leave space for form and help)
	contentHeight := m.height - 12 // Reserve space for form (8 lines) + help (4 lines)
	if contentHeight < 5 {
		contentHeight = 5
	}

	// Render the main content area
	var body string
	switch m.viewMode {
	case ViewModeSelect:
		body = m.renderModeSelect()
	case ViewModePlanning:
		body = m.renderPlanning()
	case ViewModeBuilding:
		body = m.renderBuilding()
	case ViewModeDebugging:
		body = m.renderDebugging()
	case ViewModeTechDebt:
		body = m.renderTechDebt()
	}

	// Render conversation history if we have responses
	if len(m.responses) > 0 {
		conversation := m.renderConversation(contentHeight)
		body = conversation
	}

	// Create a container for the content that takes up the available space
	contentStyle := lipgloss.NewStyle().
		Height(contentHeight).
		Width(m.width - 2).
		Padding(1)

	content := contentStyle.Render(body)

	// Render the input form at the bottom
	formView := ""
	if m.showingForm {
		formStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1).
			Width(m.width - 4)
		formView = formStyle.Render(m.form.View())
	}

	// Render help bar
	helpView := renderHelpBar()

	// Combine everything with proper spacing
	return lipgloss.JoinVertical(lipgloss.Left,
		content,
		formView,
		helpView,
	)
}

// renderConversation shows the chat history with proper scrolling
func (m Model) renderConversation(maxHeight int) string {
	if len(m.responses) == 0 {
		return m.renderTitle() + "\n\nNo conversation yet. Type a message below to get started!"
	}

	var parts []string
	parts = append(parts, m.renderTitle())
	parts = append(parts, "")

	// Calculate which responses to show based on scroll offset
	visibleResponses := m.responses[m.scrollOffset:]

	for i, resp := range visibleResponses {
		// Stop if we've filled the available height
		if len(parts) > maxHeight-5 {
			break
		}

		if resp.Answer == "" {
			// User message
			userStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#00D7FF")).
				Bold(true)
			parts = append(parts, userStyle.Render("👤 You: ")+resp.Query)
		} else {
			// Assistant message
			assistantStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7C3AED")).
				Bold(true)
			parts = append(parts, assistantStyle.Render("🤖 Assistant: "))

			// Word wrap the response
			wrapped := m.wrapText(resp.Answer, m.width-6)
			parts = append(parts, wrapped)
		}

		if i < len(visibleResponses)-1 {
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
		if len(words) == 0 {
			wrapped = append(wrapped, "")
			continue
		}

		currentLine := words[0]
		for _, word := range words[1:] {
			if len(currentLine)+1+len(word) <= width {
				currentLine += " " + word
			} else {
				wrapped = append(wrapped, currentLine)
				currentLine = word
			}
		}
		wrapped = append(wrapped, currentLine)
	}

	return strings.Join(wrapped, "\n")
}

// renderTitle is a helper for displaying the common title header
func (m Model) renderTitle() string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Render("🦆 RubrDuck - " + ModeName(m.viewMode))
}

// Run launches the TUI application
func Run(cfg *config.Config) error {
	p := tea.NewProgram(NewModel(cfg), tea.WithContext(context.Background()), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}

// generateSimulatedResponse creates a fake AI response for testing
func (m Model) generateSimulatedResponse(query string) Response {
	responses := map[ViewMode]string{
		ViewModePlanning: "Based on your request, here's a comprehensive plan:\n\n1. **Analysis Phase**\n   - Review current codebase structure\n   - Identify potential bottlenecks\n   - Assess technical debt\n\n2. **Design Phase**\n   - Create system architecture diagrams\n   - Define API contracts\n   - Plan database schema changes\n\n3. **Implementation Strategy**\n   - Break down into manageable sprints\n   - Prioritize features by business value\n   - Plan for gradual migration\n\nThis approach ensures minimal disruption while maximizing efficiency.",

		ViewModeBuilding: "I'll help you build this feature. Here's the implementation plan:\n\n```go\n// Example implementation\nfunc NewFeature() *Feature {\n    return &Feature{\n        Name: \"example\",\n        Version: \"1.0.0\",\n    }\n}\n```\n\n**Next Steps:**\n1. Create the basic structure\n2. Implement core functionality\n3. Add comprehensive tests\n4. Update documentation\n\nWould you like me to proceed with any specific part?",

		ViewModeDebugging: "Let me analyze the issue you're experiencing:\n\n**Error Analysis:**\n- Type: Runtime error\n- Location: Line 42 in main.go\n- Root cause: Nil pointer dereference\n\n**Debugging Steps:**\n1. Add logging statements around line 42\n2. Check if the variable is properly initialized\n3. Verify the data flow from upstream\n4. Test with sample data\n\n**Quick Fix:**\n```go\nif variable != nil {\n    // Safe to use variable\n}\n```\n\nLet me know if you need more specific debugging help!",

		ViewModeTechDebt: "Here's my assessment of the technical debt:\n\n**High Priority Issues:**\n- Outdated dependencies (3 packages need updates)\n- Code duplication in utils package\n- Missing error handling in 5 functions\n\n**Medium Priority:**\n- Inconsistent naming conventions\n- Long functions that need refactoring\n- Missing documentation\n\n**Recommended Actions:**\n1. Update dependencies first\n2. Refactor duplicated code\n3. Add comprehensive error handling\n4. Improve code documentation\n\nWould you like me to help with any specific refactoring?",
	}

	response, exists := responses[m.viewMode]
	if !exists {
		response = "I understand your request. Let me help you with that."
	}

	return Response{
		Mode:   m.viewMode,
		Query:  query,
		Answer: response,
		Time:   "Just now",
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

// adjustScrollToBottom positions the view so that the last response is visible.
func (m *Model) adjustScrollToBottom() {
	visible := m.height - 15 // Account for form and help areas
	if visible < 1 {
		visible = 1
	}
	m.scrollOffset = len(m.responses) - visible
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}
