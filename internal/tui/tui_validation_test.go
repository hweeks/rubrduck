package tui

import (
	"fmt"
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
	tmp, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = tmp.(Model)
	if cmd != nil {
		tmp, _ = m.Update(cmd())
		m = tmp.(Model)
	}
	assert.Len(t, m.responses, 2)
	assert.Empty(t, m.input.Value())

	// Second query: "world"
	for _, r := range "world" {
		tmp, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = tmp.(Model)
	}
	tmp, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = tmp.(Model)
	if cmd != nil {
		tmp, _ = m.Update(cmd())
		m = tmp.(Model)
	}
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

// TestMouseClickRefocus verifies that clicking anywhere focuses the input
func TestMouseClickRefocus(t *testing.T) {
	cfg := &config.Config{}
	m := NewModel(cfg)
	m.viewMode = ViewModePlanning
	m.width = 80
	m.height = 24

	// Start unfocused
	m.focused = false
	assert.False(t, m.focused)
	assert.False(t, m.input.Focused())

	// Simulate a left-click
	tmp, _ := m.Update(tea.MouseMsg{Button: tea.MouseButtonLeft})
	v := tmp.(Model)
	assert.True(t, v.focused)
	assert.True(t, v.input.Focused())
}

// TestAutoScrollOnOverflow ensures that when enough messages accumulate,
// the view auto-scrolls so the latest responses are visible.
func TestAutoScrollOnOverflow(t *testing.T) {
	cfg := &config.Config{}
	m := NewModel(cfg)
	// Use a small height so overflow occurs quickly
	m.width = 80
	m.height = 10

	// Enter Planning mode and focus input
	tmp, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = tmp.(Model)
	assert.True(t, m.focused)

	// Send many messages to overflow the visible area
	for i := 0; i < 20; i++ {
		// type a single character
		tmp, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
		m = tmp.(Model)
		// submit
		tmp, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = tmp.(Model)
		if cmd != nil {
			tmp, _ = m.Update(cmd())
			m = tmp.(Model)
		}
	}

	// After overflow, scrollOffset should be greater than zero
	assert.Greater(t, m.scrollOffset, 0)
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
		tmp, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = tmp.(Model)
		if cmd != nil {
			tmp, _ = m.Update(cmd())
			m = tmp.(Model)
		}
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

// TestScrollBounds ensures scrolling is clamped within visible window
func TestScrollBounds(t *testing.T) {
	cfg := &config.Config{}
	m := NewModel(cfg)
	m.viewMode = ViewModePlanning

	// Simulate a small terminal so visible area is limited
	m.width = 80
	m.height = 10

	// Populate responses exceeding visible capacity
	for i := 0; i < 10; i++ {
		m.responses = append(m.responses, Response{Mode: ViewModePlanning, Query: fmt.Sprintf("q%d", i), Answer: fmt.Sprintf("a%d", i), Time: "t"})
	}

	// Not focused: arrow keys should scroll
	m.focused = false

	// Scroll down past start
	for i := 0; i < 5; i++ {
		tmp, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = tmp.(Model)
	}
	// Should not exceed max offset = total-visible
	maxOff := len(m.responses) - (m.height - 6)
	assert.LessOrEqual(t, m.scrollOffset, maxOff)

	// Scroll up past zero
	for i := 0; i < 10; i++ {
		tmp, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
		m = tmp.(Model)
	}
	assert.GreaterOrEqual(t, m.scrollOffset, 0)
}
