package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hammie/rubrduck/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestInitialModel(t *testing.T) {
	cfg := &config.Config{}
	model := initialModel(cfg)

	assert.NotNil(t, model)
	assert.Equal(t, ViewModeChat, model.viewMode)
	assert.Empty(t, model.messages)
	assert.Empty(t, model.history)
	assert.False(t, model.ready)
	assert.False(t, model.quitting)
}

func TestModelInit(t *testing.T) {
	cfg := &config.Config{}
	model := initialModel(cfg)

	cmd := model.Init()
	assert.NotNil(t, cmd)
}

func TestHandleKeyPress(t *testing.T) {
	cfg := &config.Config{}
	m := initialModel(cfg)

	// Test Ctrl+C to quit
	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	newModel, cmd := m.handleKeyPress(msg)

	assert.True(t, newModel.(model).quitting)
	assert.NotNil(t, cmd)
}

func TestSubmitMessage(t *testing.T) {
	cfg := &config.Config{}
	m := initialModel(cfg)
	m.input.value = "Hello, world!"

	newModel, cmd := m.submitMessage()
	newModelTyped := newModel.(model)

	// Check that message was added
	assert.Len(t, newModelTyped.messages, 2) // User message + AI response
	assert.Equal(t, "user", newModelTyped.messages[0].role)
	assert.Equal(t, "Hello, world!", newModelTyped.messages[0].content)
	assert.Equal(t, "assistant", newModelTyped.messages[1].role)

	// Check that input was cleared
	assert.Empty(t, newModelTyped.input.value)

	// Check that it was added to history
	assert.Len(t, newModelTyped.history, 1)
	assert.Equal(t, "Hello, world!", newModelTyped.history[0])

	assert.NotNil(t, cmd)
}

func TestNavigateHistory(t *testing.T) {
	cfg := &config.Config{}
	m := initialModel(cfg)
	m.history = []string{"first", "second", "third"}

	// Test navigating up
	newModel, _ := m.navigateHistory(-1)
	newModelTyped := newModel.(model)
	assert.Equal(t, "third", newModelTyped.input.value)

	// Test navigating down
	newModel, _ = newModelTyped.navigateHistory(1)
	newModelTyped = newModel.(model)
	assert.Equal(t, "first", newModelTyped.input.value)
}

func TestSearchHistory(t *testing.T) {
	cfg := &config.Config{}
	model := initialModel(cfg)
	model.history = []string{"hello world", "goodbye world", "hello there"}

	// Test search
	results := model.searchHistory("hello")
	assert.Len(t, results, 2)
	assert.Equal(t, 0, results[0]) // "hello world"
	assert.Equal(t, 2, results[1]) // "hello there"

	// Test case insensitive search
	results = model.searchHistory("WORLD")
	assert.Len(t, results, 2)

	// Test empty search
	results = model.searchHistory("")
	assert.Empty(t, results)
}

func TestFormatContent(t *testing.T) {
	cfg := &config.Config{}
	model := initialModel(cfg)

	// Test code block formatting
	content := "```go\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```"
	formatted := model.formatContent(content)
	assert.Contains(t, formatted, "func main()")

	// Test inline code formatting
	content = "Use the `fmt.Println` function"
	formatted = model.formatContent(content)
	assert.Contains(t, formatted, "fmt.Println")

	// Test header formatting
	content = "## Header\n# Main Header"
	formatted = model.formatContent(content)
	assert.Contains(t, formatted, "Header")

	// Test list formatting
	content = "- Item 1\n- Item 2"
	formatted = model.formatContent(content)
	assert.Contains(t, formatted, "•")
}

func TestAddNewLine(t *testing.T) {
	cfg := &config.Config{}
	model := initialModel(cfg)
	model.lines = []string{"line 1", "line 2", "line 3"}
	model.currentLine = 1

	// Test adding line above
	model.addNewLineAbove()
	assert.Len(t, model.lines, 4)
	assert.Equal(t, "", model.lines[1])
	assert.Equal(t, "line 2", model.lines[2])

	// Test adding line below
	model.currentLine = 2
	model.addNewLineBelow()
	assert.Len(t, model.lines, 5)
	assert.Equal(t, "", model.lines[3])
	assert.Equal(t, 3, model.currentLine)
}

func TestViewModes(t *testing.T) {
	cfg := &config.Config{}
	m := initialModel(cfg)
	m.ready = true
	// Test chat view
	m.viewMode = ViewModeChat
	view := m.View()
	assert.Contains(t, view, "RubrDuck - AI Coding Assistant")
	// Test history view
	m.viewMode = ViewModeHistory
	view = m.View()
	assert.Contains(t, view, "Conversation History")
	// Test help view
	m.viewMode = ViewModeHelp
	view = m.View()
	assert.Contains(t, view, "Help & Keyboard Shortcuts")
}

func TestMessageRendering(t *testing.T) {
	cfg := &config.Config{}
	m := initialModel(cfg)

	msg := message{
		role:      "user",
		content:   "Hello",
		timestamp: time.Now(),
		id:        "test_1",
	}

	rendered := m.renderMessage(msg)
	assert.Contains(t, rendered, "You")
	assert.Contains(t, rendered, "Hello")

	msg.role = "assistant"
	rendered = m.renderMessage(msg)
	assert.Contains(t, rendered, "RubrDuck")
}

func TestSearchMode(t *testing.T) {
	cfg := &config.Config{}
	m := initialModel(cfg)
	m.history = []string{"hello world", "goodbye world"}

	// Test entering search mode
	msg := tea.KeyMsg{Type: tea.KeyCtrlR}
	newModel, _ := m.handleKeyPress(msg)
	newModelTyped := newModel.(model)

	assert.True(t, newModelTyped.searchMode)
	assert.Empty(t, newModelTyped.searchQuery)

	// Test search input
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
	newModel, _ = newModelTyped.handleKeyPress(msg)
	newModelTyped = newModel.(model)

	assert.Equal(t, "h", newModelTyped.searchQuery)
	assert.Len(t, newModelTyped.searchResults, 1)
}

func TestStatusMessages(t *testing.T) {
	cfg := &config.Config{}
	m := initialModel(cfg)

	// Test setting status
	cmd := m.setStatus("Test status")
	assert.NotNil(t, cmd)

	// Test status message handling
	msg := statusMsg{message: "Test status"}
	newModel, _ := m.Update(msg)
	newModelTyped := newModel.(model)

	assert.Equal(t, "Test status", newModelTyped.status)
}

func TestGenerateMockResponse(t *testing.T) {
	cfg := &config.Config{}
	m := initialModel(cfg)

	// Test hello response
	response := m.generateMockResponse("hello")
	assert.Contains(t, response, "Hello! I'm RubrDuck")

	// Test code response
	response = m.generateMockResponse("show me some code")
	assert.Contains(t, response, "```go")

	// Test help response
	response = m.generateMockResponse("help")
	assert.Contains(t, response, "Available Commands")

	// Test default response
	response = m.generateMockResponse("random message")
	assert.Contains(t, response, "mock response")
}

func TestInputAreaRendering(t *testing.T) {
	cfg := &config.Config{}
	m := initialModel(cfg)
	m.ready = true
	// Test normal input
	m.input.value = "test input"
	rendered := m.renderInputArea()
	assert.Contains(t, rendered, "test input")
	// Test search mode
	m.searchMode = true
	m.searchQuery = "search term"
	m.searchResults = []int{0, 1}
	m.history = []string{"foo", "bar"}
	rendered = m.renderInputArea()
	assert.Contains(t, rendered, "🔍 Search:")
	assert.Contains(t, rendered, "search term")
	assert.Contains(t, rendered, "(2 results)")
}

func TestHistoryViewRendering(t *testing.T) {
	cfg := &config.Config{}
	m := initialModel(cfg)

	// Test empty history
	rendered := m.renderHistoryView()
	assert.Contains(t, rendered, "No conversation history yet")

	// Test with history
	m.history = []string{"first message", "second message"}
	rendered = m.renderHistoryView()
	assert.Contains(t, rendered, "first message")
	assert.Contains(t, rendered, "second message")
}

func TestHelpViewRendering(t *testing.T) {
	cfg := &config.Config{}
	m := initialModel(cfg)

	rendered := m.renderHelpView()
	assert.Contains(t, rendered, "Keyboard Shortcuts")
	assert.Contains(t, rendered, "Ctrl+Up/Down")
	assert.Contains(t, rendered, "Ctrl+R")
	assert.Contains(t, rendered, "F1")
}

// Integration test for the complete flow
func TestCompleteTUICycle(t *testing.T) {
	cfg := &config.Config{}
	m := initialModel(cfg)

	// Simulate typing a message
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
	newModel, _ := m.Update(msg)
	m = newModel.(model)
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}}
	newModel, _ = m.Update(msg)
	m = newModel.(model)
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}
	newModel, _ = m.Update(msg)
	m = newModel.(model)
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}
	newModel, _ = m.Update(msg)
	m = newModel.(model)
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}}
	newModel, _ = m.Update(msg)
	m = newModel.(model)

	assert.Equal(t, "hello", m.input.value)

	// Simulate pressing Enter
	msg = tea.KeyMsg{Type: tea.KeyEnter}
	newModel, _ = m.Update(msg)
	m = newModel.(model)

	assert.Empty(t, m.input.value)
	assert.Len(t, m.messages, 2) // User message + AI response
	assert.Len(t, m.history, 1)
}

func TestWindowSizeHandling(t *testing.T) {
	cfg := &config.Config{}
	m := initialModel(cfg)

	// Test window size message
	msg := tea.WindowSizeMsg{Width: 80, Height: 24}
	newModel, _ := m.Update(msg)
	newModelTyped := newModel.(model)

	assert.Equal(t, 80, newModelTyped.width)
	assert.Equal(t, 24, newModelTyped.height)
	assert.True(t, newModelTyped.ready)
}

func TestQuitHandling(t *testing.T) {
	cfg := &config.Config{}
	m := initialModel(cfg)

	// Test Ctrl+C
	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	newModel, cmd := m.Update(msg)
	newModelTyped := newModel.(model)

	assert.True(t, newModelTyped.quitting)
	assert.NotNil(t, cmd)

	// Test ESC
	m = initialModel(cfg)
	msg = tea.KeyMsg{Type: tea.KeyEsc}
	newModel, cmd = m.Update(msg)
	newModelTyped = newModel.(model)

	assert.True(t, newModelTyped.quitting)
	assert.NotNil(t, cmd)
}
