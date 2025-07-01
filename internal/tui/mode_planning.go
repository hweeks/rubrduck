package tui

import (
	"fmt"
	"strings"
)

// renderPlanning renders the planning mode view
func (m Model) renderPlanning() string {
	var b strings.Builder

	if len(m.responses) > 0 {
		b.WriteString("Planning Mode - Architecture & Strategy\n")
		b.WriteString(strings.Repeat("=", 50) + "\n")

		start := m.scrollOffset
		end := start + m.height - 15
		if end > len(m.responses) {
			end = len(m.responses)
		}

		for i := start; i < end; i++ {
			response := m.responses[i]
			if response.Mode == ViewModePlanning {
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
		b.WriteString("📋 Planning Mode - Architecture & Strategy\n\n")
		b.WriteString("This mode helps you:\n")
		b.WriteString("• Design system architecture and data flows\n")
		b.WriteString("• Plan feature implementation strategies\n")
		b.WriteString("• Break down complex projects into tasks\n")
		b.WriteString("• Create development roadmaps\n\n")
		b.WriteString("What would you like to plan today?\n")
	}

	return b.String()
}
