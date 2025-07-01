package tui

import (
	"fmt"
	"strings"
)

// renderDebugging renders the debugging mode view
func (m Model) renderDebugging() string {
	var b strings.Builder

	if len(m.responses) > 0 {
		b.WriteString("Debugging Mode - Error Analysis & Problem Solving\n")
		b.WriteString(strings.Repeat("=", 50) + "\n")

		start := m.scrollOffset
		end := start + m.height - 15
		if end > len(m.responses) {
			end = len(m.responses)
		}

		for i := start; i < end; i++ {
			response := m.responses[i]
			if response.Mode == ViewModeDebugging {
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
		b.WriteString("🐛 Debugging Mode - Error Analysis & Problem Solving\n\n")
		b.WriteString("This mode helps you:\n")
		b.WriteString("• Analyze and fix runtime errors\n")
		b.WriteString("• Debug logic issues and edge cases\n")
		b.WriteString("• Trace execution flow and data\n")
		b.WriteString("• Optimize performance bottlenecks\n\n")
		b.WriteString("Describe your error or issue below!\n")
	}

	return b.String()
}
