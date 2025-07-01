package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hammie/rubrduck/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestNewModelSelectMode(t *testing.T) {
	cfg := &config.Config{}
	m := NewModel(cfg)
	assert.Equal(t, ViewModeSelect, m.viewMode)
}

func TestNewModelStartMode(t *testing.T) {
	cfg := &config.Config{TUI: config.TUIConfig{StartMode: "building"}}
	m := NewModel(cfg)
	assert.Equal(t, ViewModeBuilding, m.viewMode)
}

func TestInitReturnsCmd(t *testing.T) {
	cfg := &config.Config{}
	m := NewModel(cfg)
	cmd := m.Init()
	assert.NotNil(t, cmd)
}

func TestModeSelectView(t *testing.T) {
	cfg := &config.Config{}
	m := NewModel(cfg)
	m.width = 60
	view := m.View()
	assert.Contains(t, view, "Planning")
	assert.Contains(t, view, "Building")
	assert.Contains(t, view, "Debugging")
	assert.Contains(t, view, "Enhance")
	// first option should be selected by default
	assert.Contains(t, view, "> Planning")
}

func TestModeSelectNavigation(t *testing.T) {
	cfg := &config.Config{}
	m := NewModel(cfg)
	m.width = 60
	// Move down two times
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	v := m2.(Model)
	m2, _ = v.Update(tea.KeyMsg{Type: tea.KeyDown})
	v = m2.(Model)
	assert.Equal(t, 2, v.selectedOption)
	view := v.View()
	assert.Contains(t, view, "> Debugging")
}

func TestModeSelectEnter(t *testing.T) {
	cfg := &config.Config{}
	m := NewModel(cfg)
	m.width = 60
	// Move to Enhance
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	v := m2.(Model)
	m2, _ = v.Update(tea.KeyMsg{Type: tea.KeyDown})
	v = m2.(Model)
	m2, _ = v.Update(tea.KeyMsg{Type: tea.KeyDown})
	v = m2.(Model)
	m2, _ = v.Update(tea.KeyMsg{Type: tea.KeyEnter})
	v = m2.(Model)
	assert.Equal(t, ViewModeEnhance, v.viewMode)
}

func TestRenderPlanningView(t *testing.T) {
	cfg := &config.Config{}
	m := NewModel(cfg)
	m.width = 60
	m.height = 20
	m.viewMode = ViewModePlanning
	view := m.View()
	assert.Contains(t, view, "Planning")
	assert.Contains(t, view, "Tab")
	assert.Contains(t, view, "Navigate/Scroll")
}

func TestRenderOtherModes(t *testing.T) {
	cfg := &config.Config{}
	m := NewModel(cfg)
	m.width = 60
	m.height = 20
	m.viewMode = ViewModeBuilding
	assert.Contains(t, m.View(), "Building")
	m.viewMode = ViewModeDebugging
	assert.Contains(t, m.View(), "Debugging")
	m.viewMode = ViewModeEnhance
	assert.Contains(t, m.View(), "Enhance")
}

func TestQuitFromModes(t *testing.T) {
	cfg := &config.Config{}
	m := NewModel(cfg)
	m.viewMode = ViewModePlanning
	m.width = 60
	m.height = 20
	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	v := m2.(Model)
	assert.True(t, v.quitting)
	assert.NotNil(t, cmd)
}

func TestWindowSizeUpdates(t *testing.T) {
	cfg := &config.Config{}
	m := NewModel(cfg)
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	v := m2.(Model)
	assert.Equal(t, 80, v.width)
	assert.Equal(t, 24, v.height)
}

func TestAutoFocusOnModeSelect(t *testing.T) {
	cfg := &config.Config{}
	m := NewModel(cfg)
	m.width = 60
	m.height = 20

	// Select planning mode
	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	v := m2.(Model)

	assert.Equal(t, ViewModePlanning, v.viewMode)
	assert.True(t, v.focused)
	assert.NotNil(t, cmd)
}

func TestSimulatedResponseGeneration(t *testing.T) {
	cfg := &config.Config{}
	m := NewModel(cfg)
	m.viewMode = ViewModePlanning

	response := m.generateSimulatedResponse("test query")
	assert.Equal(t, ViewModePlanning, response.Mode)
	assert.Equal(t, "test query", response.Query)
	assert.Contains(t, response.Answer, "plan")
	assert.Equal(t, "Just now", response.Time)
}
