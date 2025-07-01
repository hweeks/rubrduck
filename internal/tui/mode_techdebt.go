package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// renderTechDebt renders the tech debt mode view
func (m Model) renderTechDebt() string {
	var b strings.Builder
	b.WriteString(m.renderTitle())
	b.WriteString("\n\n")

	// Show conversation history
	if len(m.responses) > 0 {
		b.WriteString("Tech Debt Mode - Code Quality & Refactoring\n")
		b.WriteString(strings.Repeat("=", 50) + "\n")

		// Calculate visible range for scrolling
		start := m.scrollOffset
		end := start + m.height - 15 // Leave space for input form and header
		if end > len(m.responses) {
			end = len(m.responses)
		}

		for i := start; i < end; i++ {
			response := m.responses[i]
			if response.Mode == ViewModeTechDebt {
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
		b.WriteString("🔧 Tech Debt Mode - Code Quality & Refactoring\n\n")
		b.WriteString("This mode helps you:\n")
		b.WriteString("• Identify and prioritize technical debt\n")
		b.WriteString("• Refactor legacy code and improve structure\n")
		b.WriteString("• Update dependencies and modernize code\n")
		b.WriteString("• Improve code documentation and tests\n\n")
		b.WriteString("What technical debt would you like to address?\n")
	}

	return b.String()
}

// updateTechDebt handles events in tech debt mode
func (m Model) updateTechDebt(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEscape:
			// Return to mode selection
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
