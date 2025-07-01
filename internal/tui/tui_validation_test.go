package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hammie/rubrduck/internal/config"
	"github.com/stretchr/testify/assert"
)

// TestInteractiveFlow simulates a realistic user session:
//  1. Enter Planning mode.
//  2. Submit two separate queries via the input bar.
//  3. Scroll up/down with arrow keys.
//  4. Scroll with the mouse wheel.
//
// It validates that responses are appended, input is cleared after each
// submission, and scrolling/clamping behave correctly.
func TestInteractiveFlow(t *testing.T) {
	cfg := &config.Config{}
	m := NewModel(cfg)
	m.width = 80
	m.height = 24

	// Step into Planning mode (selected by default), press Enter.
	tmp, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = tmp.(Model)
	assert.Equal(t, ViewModePlanning, m.viewMode)
	assert.True(t, m.focused)

	// Type first query: "hello" then Enter.
	for _, r := range "hello" {
		tmp, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = tmp.(Model)
	}
	tmp, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = tmp.(Model)
	assert.Len(t, m.responses, 2)
	assert.Empty(t, m.input.Value())

	// Second query: "world"
	for _, r := range "world" {
		tmp, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = tmp.(Model)
	}
	tmp, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = tmp.(Model)
	assert.Len(t, m.responses, 4)

	// Scroll up (when input not focused)
	m.focused = false
	tmp, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = tmp.(Model)
	assert.Equal(t, 0, m.scrollOffset)

	// Scroll down via arrow and mouse wheel
	for i := 0; i < 3; i++ {
		tmp, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = tmp.(Model)
	}
	tmp, _ = m.Update(tea.MouseMsg{Button: tea.MouseButtonWheelDown})
	m = tmp.(Model)
	assert.GreaterOrEqual(t, m.scrollOffset, 1)

	// Wheel up should decrement
	tmp, _ = m.Update(tea.MouseMsg{Button: tea.MouseButtonWheelUp})
	m = tmp.(Model)
	assert.GreaterOrEqual(t, m.scrollOffset, 0)
}

// Comprehensive end-to-end checks that mirror all user-requested behaviours.
func TestComprehensiveUIBehavior(t *testing.T) {
	cfg := &config.Config{}
	m := NewModel(cfg)
	m.width = 80
	m.height = 24

	// -- 1. Start in Select, choose Planning mode --
	tmp, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = tmp.(Model)
	assert.Equal(t, ViewModePlanning, m.viewMode)
	assert.True(t, m.focused)

	// -- 2. Tab toggles focus --
	tmp, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = tmp.(Model)
	assert.False(t, m.focused)
	tmp, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = tmp.(Model)
	assert.True(t, m.focused)

	// -- 3. Submit three queries and ensure responses grow --
	queries := []string{"first", "second", "third"}
	for i, q := range queries {
		for _, r := range q {
			tmp, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			m = tmp.(Model)
		}
		tmp, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = tmp.(Model)
		assert.Len(t, m.responses, (i+1)*2)
		assert.Empty(t, m.input.Value())
	}

	// -- 4. Scroll using keys --
	m.focused = false
	originalOffset := m.scrollOffset
	tmp, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = tmp.(Model)
	assert.GreaterOrEqual(t, m.scrollOffset, originalOffset)
	tmp, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = tmp.(Model)
	assert.LessOrEqual(t, m.scrollOffset, originalOffset)

	// -- 5. Scroll using mouse wheel --
	tmp, _ = m.Update(tea.MouseMsg{Button: tea.MouseButtonWheelDown})
	m = tmp.(Model)
	assert.GreaterOrEqual(t, m.scrollOffset, originalOffset)
	tmp, _ = m.Update(tea.MouseMsg{Button: tea.MouseButtonWheelUp})
	m = tmp.(Model)
	assert.LessOrEqual(t, m.scrollOffset, originalOffset+1) // small move back

	// -- 6. Window resize adjusts input width --
	tmp, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = tmp.(Model)
	assert.Equal(t, 90, m.input.Width)

	// -- 7. Escape returns to Select --
	tmp, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = tmp.(Model)
	assert.Equal(t, ViewModeSelect, m.viewMode)

	// -- 8. Choose Building mode via navigation --
	tmp, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown}) // move to Building option
	m = tmp.(Model)
	tmp, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = tmp.(Model)
	assert.Equal(t, ViewModeBuilding, m.viewMode)

	// -- 9. Quit with Ctrl+C --
	tmp, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = tmp.(Model)
	assert.True(t, m.quitting)
}
