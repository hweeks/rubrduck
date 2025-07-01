package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// renderEnhance renders the enhance mode view
func (m Model) renderEnhance() string {
	var b strings.Builder
	b.WriteString(m.renderTitle())
	b.WriteString("\n\n")

	if len(m.responses) > 0 {
		b.WriteString("Enhance Mode - Code Quality & Refactoring\n")
		b.WriteString(strings.Repeat("=", 50) + "\n")

		start := m.scrollOffset
		end := start + m.height - 15
		if end > len(m.responses) {
			end = len(m.responses)
		}

		for i := start; i < end; i++ {
			response := m.responses[i]
			if response.Mode == ViewModeEnhance {
				b.WriteString(fmt.Sprintf("\n[%s] Q: %s\n", response.Time, response.Query))
				if response.Answer != "" {
					b.WriteString("A: " + response.Answer + "\n")
				}
				b.WriteString(strings.Repeat("-", 50) + "\n")
			}
		}

		if len(m.responses) > end {
			b.WriteString(fmt.Sprintf("\n... and %d more responses (use ↑/↓ to scroll)\n", len(m.responses)-end))
		}
	} else {
		b.WriteString("🔧 Enhance Mode - Code Quality & Refactoring\n\n")
		b.WriteString("This mode helps you:\n")
		b.WriteString("• Identify and prioritize technical debt\n")
		b.WriteString("• Refactor legacy code and improve structure\n")
		b.WriteString("• Update dependencies and modernize code\n")
		b.WriteString("• Improve code documentation and tests\n\n")
		b.WriteString("What enhancements would you like to address?\n")
	}

	return b.String()
}

// updateEnhance handles events in enhance mode
func (m Model) updateEnhance(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEscape:
			m.viewMode = ViewModeSelect
		case tea.KeyUp:
			if m.scrollOffset > 0 {
				m.scrollOffset--
			}
		case tea.KeyDown:
			if m.scrollOffset < len(m.responses)-1 {
				m.scrollOffset++
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}
