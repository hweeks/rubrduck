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
func renderHelpBar() string {
	parts := []string{
		KeyStyle.Render("↑/↓"), "Navigate/Scroll",
		KeyStyle.Render("Tab"), "Focus Input",
		KeyStyle.Render("Enter"), "Select/Submit",
		KeyStyle.Render("Esc"), "Back",
		KeyStyle.Render("Ctrl+C"), "Quit",
	}
	return HelpStyle.Render(strings.Join(parts, "   "))
}
