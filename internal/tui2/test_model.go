package tui2

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/hammie/rubrduck/internal/config"
)



// NewTestModel creates a new TUI model for testing with a mock agent
func NewTestModel(cfg *config.Config, mockAgent AgentInterface) model {
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
		agent:          mockAgent, // Use mock agent for testing
	}
}

// GetViewMode returns the current view mode for testing
func (m model) GetViewMode() ViewMode {
	return m.viewMode
}

// GetSelectedOption returns the currently selected option for testing
func (m model) GetSelectedOption() int {
	return m.selectedOption
}

// GetMessages returns the current messages for testing
func (m model) GetMessages() []message {
	return m.messages
}

// GetLoading returns the current loading state for testing
func (m model) GetLoading() bool {
	return m.loading
}

// GetUserScrolling returns the current user scrolling state for testing
func (m model) GetUserScrolling() bool {
	return m.userScrolling
}

// SetViewMode sets the view mode for testing
func (m *model) SetViewMode(mode ViewMode) {
	m.viewMode = mode
}

// SetSelectedOption sets the selected option for testing
func (m *model) SetSelectedOption(option int) {
	m.selectedOption = option
}

// SetLoading sets the loading state for testing
func (m *model) SetLoading(loading bool) {
	m.loading = loading
}

// AddMessage adds a message for testing
func (m *model) AddMessage(sender, text string, mode ViewMode) {
	m.messages = append(m.messages, message{
		sender: sender,
		text:   text,
		mode:   mode,
	})
}

// ClearMessages clears all messages for testing
func (m *model) ClearMessages() {
	m.messages = make([]message, 0)
}

// GetDimensions returns the current dimensions for testing
func (m model) GetDimensions() (int, int) {
	return m.width, m.height
}

// SetDimensions sets the dimensions for testing
func (m *model) SetDimensions(width, height int) {
	m.width = width
	m.height = height
}

// GetInputValue returns the current input value for testing
func (m model) GetInputValue() string {
	return m.input.Value()
}

// SetInputValue sets the input value for testing
func (m *model) SetInputValue(value string) {
	m.input.SetValue(value)
}

// GetConfig returns the config for testing
func (m model) GetConfig() *config.Config {
	return m.config
}

// GetAgent returns the agent for testing
func (m model) GetAgent() AgentInterface {
	return m.agent
}

// RenderCurrentView renders the current view for testing
func (m model) RenderCurrentView() string {
	return m.View()
}

// RenderModeSelection renders the mode selection for testing
func (m model) RenderModeSelection() string {
	return m.renderModeSelect()
}

// RenderChatMode renders the chat mode for testing
func (m model) RenderChatMode() string {
	return m.renderChatMode()
}

// RenderChatContent renders the chat content for testing
func (m model) RenderChatContent() string {
	return m.renderChatContent()
}