package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hammie/rubrduck/internal/agent"
)

// ApprovalModel represents the approval dialog model
type ApprovalModel struct {
	request  agent.ApprovalRequest
	selected int
	approved bool
	quitting bool
	width    int
	height   int
	ready    bool
}

// ApprovalResult represents the result of an approval dialog
type ApprovalResult struct {
	Approved bool
	Reason   string
}

// Styles for approval dialog
var (
	approvalTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205")).
				Background(lipgloss.Color("236")).
				Padding(0, 1)

	riskStyle = lipgloss.NewStyle().
			Bold(true).
			Padding(0, 1)

	riskLowStyle = riskStyle.Copy().
			Foreground(lipgloss.Color("86")).
			Background(lipgloss.Color("22"))

	riskMediumStyle = riskStyle.Copy().
			Foreground(lipgloss.Color("226")).
			Background(lipgloss.Color("58"))

	riskHighStyle = riskStyle.Copy().
			Foreground(lipgloss.Color("208")).
			Background(lipgloss.Color("52"))

	riskCriticalStyle = riskStyle.Copy().
				Foreground(lipgloss.Color("196")).
				Background(lipgloss.Color("52"))

	previewStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1, 2).
			Margin(1, 0)

	buttonStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 2).
			Margin(0, 1)

	buttonSelectedStyle = buttonStyle.Copy().
				Foreground(lipgloss.Color("15")).
				Background(lipgloss.Color("205")).
				BorderForeground(lipgloss.Color("205"))

	buttonUnselectedStyle = buttonStyle.Copy().
				Foreground(lipgloss.Color("240")).
				BorderForeground(lipgloss.Color("240"))

	descriptionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Margin(1, 0)

	metadataStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true).
			Margin(1, 0)
)

// NewApprovalModel creates a new approval dialog model
func NewApprovalModel(request agent.ApprovalRequest) ApprovalModel {
	return ApprovalModel{
		request:  request,
		selected: 0, // Default to "Approve"
	}
}

// Init initializes the approval model
func (m ApprovalModel) Init() tea.Cmd {
	return tea.SetWindowTitle("RubrDuck - Approval Required")
}

// Update handles user input for the approval dialog
func (m ApprovalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.quitting = true
			return m, tea.Quit
		case tea.KeyEnter:
			m.approved = m.selected == 0 // 0 = Approve, 1 = Deny
			m.quitting = true
			return m, tea.Quit
		case tea.KeyLeft:
			if m.selected > 0 {
				m.selected--
			}
		case tea.KeyRight:
			if m.selected < 1 {
				m.selected++
			}
		case tea.KeyTab:
			m.selected = (m.selected + 1) % 2
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.ready {
			m.ready = true
		}
	}

	return m, nil
}

// View renders the approval dialog
func (m ApprovalModel) View() string {
	if m.quitting {
		return ""
	}

	if !m.ready {
		return "Initializing approval dialog..."
	}

	var s strings.Builder

	// Title
	title := approvalTitleStyle.Render("🛡️  Approval Required")
	s.WriteString(title + "\n\n")

	// Description
	description := descriptionStyle.Render(m.request.Description)
	s.WriteString(description + "\n")

	// Risk level
	riskText := m.renderRiskLevel(m.request.Risk)
	s.WriteString(riskText + "\n")

	// Preview
	if m.request.Preview != "" {
		preview := previewStyle.Render(m.request.Preview)
		s.WriteString(preview + "\n")
	}

	// Metadata
	if len(m.request.Metadata) > 0 {
		metadata := m.renderMetadata(m.request.Metadata)
		s.WriteString(metadata + "\n")
	}

	// Buttons
	buttons := m.renderButtons()
	s.WriteString(buttons + "\n")

	// Help text
	helpText := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render("Use ←/→ or Tab to navigate, Enter to confirm, Esc to cancel")

	s.WriteString(helpText)

	return s.String()
}

// renderRiskLevel renders the risk level with appropriate styling
func (m ApprovalModel) renderRiskLevel(risk agent.RiskLevel) string {
	var style lipgloss.Style
	var icon string

	switch risk {
	case agent.RiskLow:
		style = riskLowStyle
		icon = "🟢"
	case agent.RiskMedium:
		style = riskMediumStyle
		icon = "🟡"
	case agent.RiskHigh:
		style = riskHighStyle
		icon = "🟠"
	case agent.RiskCritical:
		style = riskCriticalStyle
		icon = "🔴"
	default:
		style = riskMediumStyle
		icon = "❓"
	}

	return style.Render(fmt.Sprintf("%s Risk Level: %s", icon, strings.ToUpper(string(risk))))
}

// renderMetadata renders metadata information
func (m ApprovalModel) renderMetadata(metadata map[string]interface{}) string {
	var s strings.Builder
	s.WriteString("Details:\n")

	for key, value := range metadata {
		s.WriteString(fmt.Sprintf("  %s: %v\n", key, value))
	}

	return metadataStyle.Render(s.String())
}

// renderButtons renders the approve/deny buttons
func (m ApprovalModel) renderButtons() string {
	var s strings.Builder

	approveText := "✅ Approve"
	denyText := "❌ Deny"

	if m.selected == 0 {
		s.WriteString(buttonSelectedStyle.Render(approveText))
		s.WriteString(" ")
		s.WriteString(buttonUnselectedStyle.Render(denyText))
	} else {
		s.WriteString(buttonUnselectedStyle.Render(approveText))
		s.WriteString(" ")
		s.WriteString(buttonSelectedStyle.Render(denyText))
	}

	return s.String()
}

// GetResult returns the approval result
func (m ApprovalModel) GetResult() ApprovalResult {
	if m.approved {
		return ApprovalResult{
			Approved: true,
			Reason:   "User approved",
		}
	}
	return ApprovalResult{
		Approved: false,
		Reason:   "User denied",
	}
}

// ShowApprovalDialog displays an approval dialog and returns the result
func ShowApprovalDialog(request agent.ApprovalRequest) (ApprovalResult, error) {
	model := NewApprovalModel(request)
	program := tea.NewProgram(model, tea.WithAltScreen())

	_, err := program.Run()
	if err != nil {
		return ApprovalResult{Approved: false, Reason: fmt.Sprintf("Dialog error: %v", err)}, err
	}

	return model.GetResult(), nil
}

// BatchApprovalModel represents a batch approval dialog model
type BatchApprovalModel struct {
	requests   []agent.ApprovalRequest
	selected   int
	approved   bool
	quitting   bool
	width      int
	height     int
	ready      bool
	scrollY    int
	maxScrollY int
}

// NewBatchApprovalModel creates a new batch approval dialog model
func NewBatchApprovalModel(requests []agent.ApprovalRequest) BatchApprovalModel {
	return BatchApprovalModel{
		requests: requests,
		selected: 0,
	}
}

// Init initializes the batch approval model
func (m BatchApprovalModel) Init() tea.Cmd {
	return tea.SetWindowTitle("RubrDuck - Batch Approval Required")
}

// Update handles user input for the batch approval dialog
func (m BatchApprovalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.quitting = true
			return m, tea.Quit
		case tea.KeyEnter:
			m.approved = m.selected == 0 // 0 = Approve All, 1 = Deny All
			m.quitting = true
			return m, tea.Quit
		case tea.KeyUp:
			if m.selected > 0 {
				m.selected--
			}
		case tea.KeyDown:
			if m.selected < 1 {
				m.selected++
			}
		case tea.KeyLeft:
			if m.scrollY > 0 {
				m.scrollY--
			}
		case tea.KeyRight:
			if m.scrollY < m.maxScrollY {
				m.scrollY++
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.ready {
			m.ready = true
			// Calculate max scroll based on content height
			m.maxScrollY = len(m.requests) * 3 // Rough estimate
		}
	}

	return m, nil
}

// View renders the batch approval dialog
func (m BatchApprovalModel) View() string {
	if m.quitting {
		return ""
	}

	if !m.ready {
		return "Initializing batch approval dialog..."
	}

	var s strings.Builder

	// Title
	title := approvalTitleStyle.Render(fmt.Sprintf("🛡️  Batch Approval Required (%d operations)", len(m.requests)))
	s.WriteString(title + "\n\n")

	// Operations list
	operations := m.renderOperationsList()
	s.WriteString(operations + "\n")

	// Buttons
	buttons := m.renderBatchButtons()
	s.WriteString(buttons + "\n")

	// Help text
	helpText := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Render("Use ↑/↓ to navigate buttons, ←/→ to scroll operations, Enter to confirm, Esc to cancel")

	s.WriteString(helpText)

	return s.String()
}

// renderOperationsList renders the list of operations
func (m BatchApprovalModel) renderOperationsList() string {
	var s strings.Builder
	s.WriteString("Operations to be executed:\n\n")

	// Calculate visible range based on scroll
	start := m.scrollY
	end := start + 10 // Show max 10 operations at a time
	if end > len(m.requests) {
		end = len(m.requests)
	}

	for i := start; i < end; i++ {
		req := m.requests[i]
		riskText := m.renderRiskLevel(req.Risk)
		s.WriteString(fmt.Sprintf("%d. %s\n   %s\n\n", i+1, req.Description, riskText))
	}

	if len(m.requests) > 10 {
		s.WriteString(fmt.Sprintf("... and %d more operations\n", len(m.requests)-10))
	}

	return previewStyle.Render(s.String())
}

// renderRiskLevel renders the risk level for batch operations
func (m BatchApprovalModel) renderRiskLevel(risk agent.RiskLevel) string {
	var icon string

	switch risk {
	case agent.RiskLow:
		icon = "🟢"
	case agent.RiskMedium:
		icon = "🟡"
	case agent.RiskHigh:
		icon = "🟠"
	case agent.RiskCritical:
		icon = "🔴"
	default:
		icon = "❓"
	}

	return fmt.Sprintf("%s %s risk", icon, strings.ToUpper(string(risk)))
}

// renderBatchButtons renders the batch approve/deny buttons
func (m BatchApprovalModel) renderBatchButtons() string {
	var s strings.Builder

	approveText := "✅ Approve All"
	denyText := "❌ Deny All"

	if m.selected == 0 {
		s.WriteString(buttonSelectedStyle.Render(approveText))
		s.WriteString(" ")
		s.WriteString(buttonUnselectedStyle.Render(denyText))
	} else {
		s.WriteString(buttonUnselectedStyle.Render(approveText))
		s.WriteString(" ")
		s.WriteString(buttonSelectedStyle.Render(denyText))
	}

	return s.String()
}

// GetResult returns the batch approval result
func (m BatchApprovalModel) GetResult() ApprovalResult {
	if m.approved {
		return ApprovalResult{
			Approved: true,
			Reason:   "User approved all operations",
		}
	}
	return ApprovalResult{
		Approved: false,
		Reason:   "User denied all operations",
	}
}

// ShowBatchApprovalDialog displays a batch approval dialog and returns the result
func ShowBatchApprovalDialog(requests []agent.ApprovalRequest) (ApprovalResult, error) {
	model := NewBatchApprovalModel(requests)
	program := tea.NewProgram(model, tea.WithAltScreen())

	_, err := program.Run()
	if err != nil {
		return ApprovalResult{Approved: false, Reason: fmt.Sprintf("Dialog error: %v", err)}, err
	}

	return model.GetResult(), nil
}
