package testing

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hammie/rubrduck/internal/config"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestTUI provides a comprehensive testing framework for TUI applications
type TestTUI struct {
	t        *testing.T
	program  *tea.Program
	model    tea.Model
	mockAgent *MockAgent
	config    *config.Config
	width     int
	height    int
	messages  []tea.Msg
	outputs   []string
}

// AgentInterface defines the interface that agents must implement for testing
type AgentInterface interface {
	Chat(ctx context.Context, message string) (string, error)
	ClearHistory()
}

// MockAgent provides a mock implementation of the agent interface
type MockAgent struct {
	mock.Mock
}

func (m *MockAgent) Chat(ctx context.Context, message string) (string, error) {
	args := m.Called(ctx, message)
	return args.String(0), args.Error(1)
}

func (m *MockAgent) ClearHistory() {
	m.Called()
}

// TUIModel defines the interface for TUI models that can be tested
type TUIModel interface {
	tea.Model
	// Additional methods for testing
	GetViewMode() interface{}
	GetMessages() []interface{}
	GetLoading() bool
	SetViewMode(mode interface{})
	AddMessage(sender, text string, mode interface{})
}

// NewTestTUI creates a new TUI testing instance
func NewTestTUI(t *testing.T, opts ...TestOption) *TestTUI {
	testTUI := &TestTUI{
		t:      t,
		width:  80,
		height: 24,
		config: &config.Config{
			Model: "test-model",
		},
	}

	// Apply options
	for _, opt := range opts {
		opt(testTUI)
	}

	// Create mock agent
	testTUI.mockAgent = &MockAgent{}

	// Setup color profile for consistent testing
	lipgloss.SetColorProfile(termenv.Ascii)

	return testTUI
}

// TestOption configures the test TUI
type TestOption func(*TestTUI)

// WithSize sets the terminal size for testing
func WithSize(width, height int) TestOption {
	return func(tt *TestTUI) {
		tt.width = width
		tt.height = height
	}
}

// WithConfig sets the config for testing
func WithConfig(cfg *config.Config) TestOption {
	return func(tt *TestTUI) {
		tt.config = cfg
	}
}

// Start initializes the TUI with the test model
func (tt *TestTUI) Start(model tea.Model) {
	tt.model = model
	tt.outputs = []string{}
	tt.messages = []tea.Msg{}
	
	// Initialize the model
	if initCmd := model.Init(); initCmd != nil {
		// Handle initial commands if needed
		tt.messages = append(tt.messages, tea.WindowSizeMsg{
			Width:  tt.width,
			Height: tt.height,
		})
	}
}

// StartWithMockAgent starts the TUI with a mock agent
func (tt *TestTUI) StartWithMockAgent() {
	// For now, we'll create a simple model that can be used for testing
	// This avoids the import cycle by not importing tui2 directly
	tt.Start(tt.createTestModel())
}

// createTestModel creates a basic test model for testing purposes
func (tt *TestTUI) createTestModel() tea.Model {
	// This is a simplified model for testing
	return &basicTestModel{
		config:    tt.config,
		mockAgent: tt.mockAgent,
		width:     tt.width,
		height:    tt.height,
	}
}

// basicTestModel is a simple model used for testing
type basicTestModel struct {
	config    *config.Config
	mockAgent *MockAgent
	width     int
	height    int
	viewMode  string
	messages  []testMessage
	loading   bool
}

type testMessage struct {
	sender string
	text   string
	mode   string
}

func (m *basicTestModel) Init() tea.Cmd {
	return nil
}

func (m *basicTestModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			if m.viewMode == "select" {
				m.viewMode = "planning"
			}
		case "escape":
			if m.viewMode != "select" {
				m.viewMode = "select"
			}
		}
	}
	return m, nil
}

func (m *basicTestModel) View() string {
	if m.viewMode == "select" || m.viewMode == "" {
		return "🦆 RubrDuck - Choose Your Mode\n\n📋 Planning - Architecture design and project planning\n🔨 Building - Code implementation and development\n🐛 Debugging - Error analysis and problem solving\n🔧 Enhance - Code quality improvement and refactoring\n\nUse ↑/↓ to navigate, Enter to select, Ctrl+C to exit"
	}
	return fmt.Sprintf("📋 Planning Mode (ESC to return to mode selection)\n\nYou:   Test message\nAI:    Test response\n\n❯ ")
}

func (m *basicTestModel) GetViewMode() interface{} {
	return m.viewMode
}

func (m *basicTestModel) GetMessages() []interface{} {
	msgs := make([]interface{}, len(m.messages))
	for i, msg := range m.messages {
		msgs[i] = msg
	}
	return msgs
}

func (m *basicTestModel) GetLoading() bool {
	return m.loading
}

func (m *basicTestModel) SetViewMode(mode interface{}) {
	if s, ok := mode.(string); ok {
		m.viewMode = s
	}
}

func (m *basicTestModel) AddMessage(sender, text string, mode interface{}) {
	m.messages = append(m.messages, testMessage{
		sender: sender,
		text:   text,
		mode:   mode.(string),
	})
}

// SendKey sends a key press to the TUI
func (tt *TestTUI) SendKey(key tea.KeyMsg) {
	tt.messages = append(tt.messages, key)
	newModel, _ := tt.model.Update(key)
	tt.model = newModel
	tt.outputs = append(tt.outputs, tt.model.View())
}

// SendKeys sends multiple key presses
func (tt *TestTUI) SendKeys(keys ...tea.KeyMsg) {
	for _, key := range keys {
		tt.SendKey(key)
	}
}

// SendString sends a string as key presses
func (tt *TestTUI) SendString(s string) {
	for _, r := range s {
		tt.SendKey(tea.KeyMsg{
			Type:  tea.KeyRunes,
			Runes: []rune{r},
		})
	}
}

// SendEnter sends an Enter key press
func (tt *TestTUI) SendEnter() {
	tt.SendKey(tea.KeyMsg{Type: tea.KeyEnter})
}

// SendEscape sends an Escape key press
func (tt *TestTUI) SendEscape() {
	tt.SendKey(tea.KeyMsg{Type: tea.KeyEsc})
}

// SendCtrlC sends a Ctrl+C key press
func (tt *TestTUI) SendCtrlC() {
	tt.SendKey(tea.KeyMsg{Type: tea.KeyCtrlC})
}

// SendArrowUp sends an up arrow key press
func (tt *TestTUI) SendArrowUp() {
	tt.SendKey(tea.KeyMsg{Type: tea.KeyUp})
}

// SendArrowDown sends a down arrow key press
func (tt *TestTUI) SendArrowDown() {
	tt.SendKey(tea.KeyMsg{Type: tea.KeyDown})
}

// SendMessage sends any tea.Msg to the TUI
func (tt *TestTUI) SendMessage(msg tea.Msg) {
	tt.messages = append(tt.messages, msg)
	newModel, _ := tt.model.Update(msg)
	tt.model = newModel
	tt.outputs = append(tt.outputs, tt.model.View())
}

// WaitForOutput waits for specific output to appear
func (tt *TestTUI) WaitForOutput(expected string, timeout time.Duration) {
	start := time.Now()
	for time.Since(start) < timeout {
		if tt.currentOutputContains(expected) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	tt.t.Errorf("Expected output %q not found within timeout", expected)
}

// WaitForOutputPattern waits for output matching a pattern
func (tt *TestTUI) WaitForOutputPattern(pattern string, timeout time.Duration) {
	start := time.Now()
	for time.Since(start) < timeout {
		if tt.currentOutputContains(pattern) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	tt.t.Errorf("Expected output pattern %q not found within timeout", pattern)
}

// GetFinalOutput returns the final output of the TUI
func (tt *TestTUI) GetFinalOutput() []byte {
	if len(tt.outputs) == 0 {
		return []byte(tt.model.View())
	}
	return []byte(tt.outputs[len(tt.outputs)-1])
}

// GetCurrentOutput returns the current output without waiting for program to finish
func (tt *TestTUI) GetCurrentOutput() []byte {
	return []byte(tt.model.View())
}

// currentOutputContains checks if current output contains the expected string
func (tt *TestTUI) currentOutputContains(expected string) bool {
	current := tt.model.View()
	return strings.Contains(current, expected)
}

// Quit terminates the TUI
func (tt *TestTUI) Quit() {
	tt.SendKey(tea.KeyMsg{Type: tea.KeyCtrlC})
}

// QuitWithTimeout terminates the TUI with a specific timeout
func (tt *TestTUI) QuitWithTimeout(timeout time.Duration) {
	tt.SendKey(tea.KeyMsg{Type: tea.KeyCtrlC})
}

// MockAgent returns the mock agent for setting expectations
func (tt *TestTUI) MockAgent() *MockAgent {
	return tt.mockAgent
}

// AssertOutput asserts that the output contains expected content
func (tt *TestTUI) AssertOutput(expected string) {
	output := tt.GetCurrentOutput()
	assert.Contains(tt.t, string(output), expected, "Output should contain expected content")
}

// AssertOutputNotContains asserts that the output does not contain content
func (tt *TestTUI) AssertOutputNotContains(unexpected string) {
	output := tt.GetCurrentOutput()
	assert.NotContains(tt.t, string(output), unexpected, "Output should not contain unexpected content")
}

// AssertOutputLines asserts specific lines in the output
func (tt *TestTUI) AssertOutputLines(expectedLines ...string) {
	output := string(tt.GetCurrentOutput())
	lines := strings.Split(output, "\n")
	
	for _, expectedLine := range expectedLines {
		found := false
		for _, line := range lines {
			if strings.Contains(line, expectedLine) {
				found = true
				break
			}
		}
		assert.True(tt.t, found, "Expected line not found: %s", expectedLine)
	}
}

// AssertModeSelection asserts that the mode selection screen is displayed
func (tt *TestTUI) AssertModeSelection() {
	tt.AssertOutput("🦆 RubrDuck - Choose Your Mode")
	tt.AssertOutput("📋 Planning - Architecture design and project planning")
	tt.AssertOutput("🔨 Building - Code implementation and development")
	tt.AssertOutput("🐛 Debugging - Error analysis and problem solving")
	tt.AssertOutput("🔧 Enhance - Code quality improvement and refactoring")
}

// AssertModeSelected asserts that a specific mode is selected
func (tt *TestTUI) AssertModeSelected(modeIndex int) {
	modes := []string{"Planning", "Building", "Debugging", "Enhance"}
	icons := []string{"📋", "🔨", "🐛", "🔧"}
	
	require.Less(tt.t, modeIndex, len(modes), "Mode index out of range")
	
	// Check that the selected mode has the selection indicator
	expectedLine := fmt.Sprintf("❯ %s %s", icons[modeIndex], modes[modeIndex])
	tt.AssertOutput(expectedLine)
}

// AssertChatMode asserts that the chat mode is active for a specific mode
func (tt *TestTUI) AssertChatMode(modeName string) {
	tt.AssertOutput(fmt.Sprintf("%s Mode (ESC to return to mode selection)", modeName))
}

// AssertUserMessage asserts that a user message is displayed
func (tt *TestTUI) AssertUserMessage(message string) {
	tt.AssertOutput("You:   " + message)
}

// AssertAIMessage asserts that an AI message is displayed
func (tt *TestTUI) AssertAIMessage(message string) {
	tt.AssertOutput("AI:    " + message)
}

// AssertLoadingIndicator asserts that the loading indicator is shown
func (tt *TestTUI) AssertLoadingIndicator() {
	tt.AssertOutput("AI thinking...")
}

// AssertWelcomeMessage asserts that the welcome message for a mode is shown
func (tt *TestTUI) AssertWelcomeMessage(mode string) {
	welcomeMessages := map[string]string{
		"Planning":  "Planning Mode - Let's design your system architecture",
		"Building":  "Building Mode - Time to implement features",
		"Debugging": "Debugging Mode - Let's analyze errors",
		"Enhance":   "Enhance Mode - Improve code quality",
	}
	
	if expected, ok := welcomeMessages[mode]; ok {
		tt.AssertOutput(expected)
	}
}

// Scenario represents a test scenario
type Scenario struct {
	Name        string
	Description string
	Steps       []Step
}

// Step represents a test step
type Step struct {
	Name     string
	Action   func(*TestTUI)
	Validate func(*TestTUI)
}

// RunScenario runs a complete test scenario
func (tt *TestTUI) RunScenario(scenario Scenario) {
	tt.t.Run(scenario.Name, func(t *testing.T) {
		for i, step := range scenario.Steps {
			stepName := fmt.Sprintf("Step %d: %s", i+1, step.Name)
			tt.t.Run(stepName, func(t *testing.T) {
				if step.Action != nil {
					step.Action(tt)
				}
				if step.Validate != nil {
					step.Validate(tt)
				}
			})
		}
	})
}

// Common key messages for testing
var (
	KeyEnter  = tea.KeyMsg{Type: tea.KeyEnter}
	KeyEsc    = tea.KeyMsg{Type: tea.KeyEsc}
	KeyCtrlC  = tea.KeyMsg{Type: tea.KeyCtrlC}
	KeyUp     = tea.KeyMsg{Type: tea.KeyUp}
	KeyDown   = tea.KeyMsg{Type: tea.KeyDown}
	KeyLeft   = tea.KeyMsg{Type: tea.KeyLeft}
	KeyRight  = tea.KeyMsg{Type: tea.KeyRight}
	KeySpace  = tea.KeyMsg{Type: tea.KeySpace}
	KeyTab    = tea.KeyMsg{Type: tea.KeyTab}
	KeyBackspace = tea.KeyMsg{Type: tea.KeyBackspace}
)

// Helper function to create key messages for characters
func KeyRune(r rune) tea.KeyMsg {
	return tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{r},
	}
}

// Helper function to create key messages for strings
func KeyString(s string) []tea.KeyMsg {
	keys := make([]tea.KeyMsg, len(s))
	for i, r := range s {
		keys[i] = KeyRune(r)
	}
	return keys
}