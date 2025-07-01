package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// renderModeSelect renders the mode selection interface
func (m Model) renderModeSelect() string {
	var b strings.Builder
	b.WriteString(m.renderTitle())
	b.WriteString("\n\n")
	b.WriteString("Choose your RubrDuck mode:\n\n")

	// List available modes with descriptions
	modes := []struct {
		name        string
		mode        ViewMode
		description string
		icon        string
	}{
		{"Planning", ViewModePlanning, "Architecture design and project planning", "📋"},
		{"Building", ViewModeBuilding, "Code implementation and development", "🔨"},
		{"Debugging", ViewModeDebugging, "Error analysis and problem solving", "🐛"},
		{"Tech Debt", ViewModeTechDebt, "Code quality improvement and refactoring", "🔧"},
	}

	for i, modeInfo := range modes {
		prefix := "  "
		if i == m.selectedOption {
			prefix = "▶ "
		}

		style := lipgloss.NewStyle().
			Foreground(textPrimary)

		if i == m.selectedOption {
			style = style.
				Bold(true).
				Foreground(primaryColor)
		}

		line := style.Render(prefix + modeInfo.icon + " " + modeInfo.name + " - " + modeInfo.description)
		b.WriteString(line + "\n")
	}

	b.WriteString("\nUse ↑/↓ to navigate, Enter to select a mode")

	return b.String()
}

// updateModeSelect handles events in mode selection
func (m Model) updateModeSelect(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyUp:
			if m.selectedOption > 0 {
				m.selectedOption--
			}
		case tea.KeyDown:
			if m.selectedOption < 3 { // 4 total modes (0-3)
				m.selectedOption++
			}
		case tea.KeyEnter:
			// Switch to selected mode
			switch m.selectedOption {
			case 0:
				m.viewMode = ViewModePlanning
			case 1:
				m.viewMode = ViewModeBuilding
			case 2:
				m.viewMode = ViewModeDebugging
			case 3:
				m.viewMode = ViewModeTechDebt
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}
