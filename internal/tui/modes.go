package tui

// ViewMode represents one of the TUI modes
type ViewMode int

const (
	// ViewModeSelect prompts the user to select a mode
	ViewModeSelect ViewMode = iota
	// ViewModePlanning is the planning screen
	ViewModePlanning
	// ViewModeBuilding is the building screen
	ViewModeBuilding
	// ViewModeDebugging is the debugging screen
	ViewModeDebugging
	// ViewModeTechDebt is the tech debt screen
	ViewModeTechDebt
)

// ModeOptions enumerates the available modes in selection order
var ModeOptions = []ViewMode{
	ViewModePlanning,
	ViewModeBuilding,
	ViewModeDebugging,
	ViewModeTechDebt,
}

// ModeName returns a human-friendly name for a ViewMode
func ModeName(mode ViewMode) string {
	switch mode {
	case ViewModePlanning:
		return "Planning"
	case ViewModeBuilding:
		return "Building"
	case ViewModeDebugging:
		return "Debugging"
	case ViewModeTechDebt:
		return "Tech Debt"
	default:
		return ""
	}
}
