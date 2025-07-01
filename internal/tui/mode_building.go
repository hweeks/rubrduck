package tui

import (
   "fmt"
   "strings"

   tea "github.com/charmbracelet/bubbletea"
)

// renderBuilding renders the building mode view
func (m Model) renderBuilding() string {
   var b strings.Builder
   b.WriteString(m.renderTitle())
   b.WriteString("\n\n")

   if len(m.responses) > 0 {
       b.WriteString("Building Mode - Code Implementation & Development\n")
       b.WriteString(strings.Repeat("=", 50) + "\n")

       start := m.scrollOffset
       end := start + m.height - 15
       if end > len(m.responses) {
           end = len(m.responses)
       }

       for i := start; i < end; i++ {
           response := m.responses[i]
           if response.Mode == ViewModeBuilding {
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
       b.WriteString("🔨 Building Mode - Code Implementation & Development\n\n")
       b.WriteString("This mode helps you:\n")
       b.WriteString("• Implement new features and functionality\n")
       b.WriteString("• Generate code examples and boilerplate\n")
       b.WriteString("• Refactor and optimize existing code\n")
       b.WriteString("• Create tests and documentation\n\n")
       b.WriteString("Start by typing your development question below!\n")
   }

   return b.String()
}

// updateBuilding handles events in building mode
func (m Model) updateBuilding(msg tea.Msg) (Model, tea.Cmd) {
   switch msg := msg.(type) {
   case tea.KeyMsg:
       switch msg.Type {
       case tea.KeyEscape:
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