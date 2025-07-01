package tui

import (
	"fmt"
	"strings"
)

// renderEnhance renders the enhance mode view
func (m Model) renderEnhance() string {
	var b strings.Builder

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
