package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Color palette using adaptive colors
var (
	primaryColor = lipgloss.AdaptiveColor{Light: "#7D56F4", Dark: "#BD93F9"}
	textMuted    = lipgloss.AdaptiveColor{Light: "#ADB5BD", Dark: "#6272A4"}
	textPrimary  = lipgloss.AdaptiveColor{Light: "#212529", Dark: "#F8F8F2"}
	bgPrimary    = lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#282A36"}
	bgSecondary  = lipgloss.AdaptiveColor{Light: "#F8F9FA", Dark: "#44475A"}
)

// Style definitions for the TUI layout
var (
	// AppStyle is the root container style
	AppStyle = lipgloss.NewStyle().
			Padding(1, 2).
			Background(bgPrimary)

	// DividerStyle renders a horizontal divider
	DividerStyle = lipgloss.NewStyle().
			Foreground(textMuted).
			Margin(1, 0)

	// Input styles for prompt and user input
	InputStyle = lipgloss.NewStyle().
			Background(bgSecondary).
			Foreground(textPrimary).
			Padding(0, 2)

	// InputFocusedStyle highlights the input field
	InputFocusedStyle = InputStyle.Copy().
				BorderLeft(true).
				BorderLeftForeground(primaryColor)

	// PromptStyle styles the input prompt
	PromptStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(textPrimary).
			MarginRight(1)

	// HelpStyle renders the help text at the bottom
	HelpStyle = lipgloss.NewStyle().
			Foreground(textMuted).
			Italic(true).
			MarginTop(1)

	// KeyStyle styles key hints within help text
	KeyStyle = lipgloss.NewStyle().
			Background(bgSecondary).
			Foreground(textPrimary).
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(textMuted)
)

// renderHelpBar renders key hints at the bottom of the UI
// renderStatusBar renders the bottom status bar: help hints on left and mode info on right
func renderStatusBar(m Model) string {
	// Help hints (flat text)
	hints := []string{
		"↑/↓ Navigate/Scroll",
		"Tab Focus Input",
		"Enter Submit",
		"Esc Back",
		"Ctrl+C Quit",
	}
	help := strings.Join(hints, "   ")
	// Current mode on right
	info := ModeName(m.viewMode)

	// Pad between help and mode info
	totalWidth := m.width - 4
	pad := totalWidth - lipgloss.Width(help) - lipgloss.Width(info)
	if pad < 1 {
		pad = 1
	}
	line := help + strings.Repeat(" ", pad) + info
	return HelpStyle.Render(line)
}
