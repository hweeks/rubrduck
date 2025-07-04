package testing

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/mock"
)

// PredefinedScenarios contains common test scenarios for the TUI
var PredefinedScenarios = struct {
	InitialDisplay        Scenario
	ModeNavigation        Scenario
	ModeSelection         Scenario
	ChatInteraction       Scenario
	AIResponseHandling    Scenario
	ErrorHandling         Scenario
	NavigationBackToModes Scenario
	KeyboardShortcuts     Scenario
	ResizeHandling        Scenario
}{
	InitialDisplay: Scenario{
		Name:        "Initial Display",
		Description: "Test that the TUI displays correctly on startup",
		Steps: []Step{
			{
				Name: "Initialize TUI",
				Action: func(tt *TestTUI) {
					tt.StartWithMockAgent()
				},
				Validate: func(tt *TestTUI) {
					tt.AssertModeSelection()
					tt.AssertModeSelected(0) // Planning should be selected by default
				},
			},
		},
	},

	ModeNavigation: Scenario{
		Name:        "Mode Navigation",
		Description: "Test navigation between different modes in the selection screen",
		Steps: []Step{
			{
				Name: "Initialize TUI",
				Action: func(tt *TestTUI) {
					tt.StartWithMockAgent()
				},
				Validate: func(tt *TestTUI) {
					tt.AssertModeSelection()
					tt.AssertModeSelected(0)
				},
			},
			{
				Name: "Navigate down to Building mode",
				Action: func(tt *TestTUI) {
					tt.SendArrowDown()
				},
				Validate: func(tt *TestTUI) {
					tt.AssertModeSelected(1)
				},
			},
			{
				Name: "Navigate down to Debugging mode",
				Action: func(tt *TestTUI) {
					tt.SendArrowDown()
				},
				Validate: func(tt *TestTUI) {
					tt.AssertModeSelected(2)
				},
			},
			{
				Name: "Navigate down to Enhance mode",
				Action: func(tt *TestTUI) {
					tt.SendArrowDown()
				},
				Validate: func(tt *TestTUI) {
					tt.AssertModeSelected(3)
				},
			},
			{
				Name: "Navigate up to Debugging mode",
				Action: func(tt *TestTUI) {
					tt.SendArrowUp()
				},
				Validate: func(tt *TestTUI) {
					tt.AssertModeSelected(2)
				},
			},
		},
	},

	ModeSelection: Scenario{
		Name:        "Mode Selection",
		Description: "Test selecting a mode and entering chat interface",
		Steps: []Step{
			{
				Name: "Initialize TUI",
				Action: func(tt *TestTUI) {
					tt.StartWithMockAgent()
				},
				Validate: func(tt *TestTUI) {
					tt.AssertModeSelection()
				},
			},
			{
				Name: "Select Planning mode",
				Action: func(tt *TestTUI) {
					tt.SendEnter()
				},
				Validate: func(tt *TestTUI) {
					tt.AssertChatMode("📋 Planning")
					tt.AssertWelcomeMessage("Planning")
				},
			},
		},
	},

	ChatInteraction: Scenario{
		Name:        "Chat Interaction",
		Description: "Test sending messages and receiving AI responses",
		Steps: []Step{
			{
				Name: "Initialize TUI and enter Planning mode",
				Action: func(tt *TestTUI) {
					tt.StartWithMockAgent()
					tt.SendEnter() // Select Planning mode
				},
				Validate: func(tt *TestTUI) {
					tt.AssertChatMode("📋 Planning")
				},
			},
			{
				Name: "Send user message",
				Action: func(tt *TestTUI) {
					// Setup mock response
					tt.MockAgent().On("Chat", mock.Anything, "Design a web application").
						Return("I can help you design a web application. Let's start with the architecture...", nil)
					
					tt.SendString("Design a web application")
					tt.SendEnter()
				},
				Validate: func(tt *TestTUI) {
					tt.AssertUserMessage("Design a web application")
					tt.AssertLoadingIndicator()
				},
			},
			{
				Name: "Wait for AI response",
				Action: func(tt *TestTUI) {
					tt.WaitForOutput("I can help you design a web application", 5*time.Second)
				},
				Validate: func(tt *TestTUI) {
					tt.AssertAIMessage("I can help you design a web application")
				},
			},
		},
	},

	AIResponseHandling: Scenario{
		Name:        "AI Response Handling",
		Description: "Test different AI response scenarios including success and error cases",
		Steps: []Step{
			{
				Name: "Initialize TUI and enter Building mode",
				Action: func(tt *TestTUI) {
					tt.StartWithMockAgent()
					tt.SendArrowDown() // Navigate to Building
					tt.SendEnter()
				},
				Validate: func(tt *TestTUI) {
					tt.AssertChatMode("🔨 Building")
				},
			},
			{
				Name: "Test successful AI response",
				Action: func(tt *TestTUI) {
					tt.MockAgent().On("Chat", mock.Anything, "Create a function").
						Return("Here's a function for you:\n\nfunc example() {\n  // implementation\n}", nil)
					
					tt.SendString("Create a function")
					tt.SendEnter()
				},
				Validate: func(tt *TestTUI) {
					tt.WaitForOutput("Here's a function for you", 5*time.Second)
					tt.AssertAIMessage("Here's a function for you")
				},
			},
			{
				Name: "Test AI error response",
				Action: func(tt *TestTUI) {
					tt.MockAgent().On("Chat", mock.Anything, "Invalid request").
						Return("", context.DeadlineExceeded)
					
					tt.SendString("Invalid request")
					tt.SendEnter()
				},
				Validate: func(tt *TestTUI) {
					tt.WaitForOutput("❌ Error:", 5*time.Second)
					tt.AssertOutput("❌ Error: context deadline exceeded")
				},
			},
		},
	},

	ErrorHandling: Scenario{
		Name:        "Error Handling",
		Description: "Test error scenarios and error message display",
		Steps: []Step{
			{
				Name: "Initialize TUI and enter Debugging mode",
				Action: func(tt *TestTUI) {
					tt.StartWithMockAgent()
					tt.SendArrowDown() // Navigate to Building
					tt.SendArrowDown() // Navigate to Debugging
					tt.SendEnter()
				},
				Validate: func(tt *TestTUI) {
					tt.AssertChatMode("🐛 Debugging")
				},
			},
			{
				Name: "Test network error",
				Action: func(tt *TestTUI) {
					tt.MockAgent().On("Chat", mock.Anything, "Debug this error").
						Return("", context.DeadlineExceeded)
					
					tt.SendString("Debug this error")
					tt.SendEnter()
				},
				Validate: func(tt *TestTUI) {
					tt.WaitForOutput("❌ Error:", 5*time.Second)
					tt.AssertOutput("context deadline exceeded")
				},
			},
		},
	},

	NavigationBackToModes: Scenario{
		Name:        "Navigation Back to Modes",
		Description: "Test returning to mode selection from chat interface",
		Steps: []Step{
			{
				Name: "Initialize TUI and enter a mode",
				Action: func(tt *TestTUI) {
					tt.StartWithMockAgent()
					tt.SendEnter() // Select Planning mode
				},
				Validate: func(tt *TestTUI) {
					tt.AssertChatMode("📋 Planning")
				},
			},
			{
				Name: "Return to mode selection",
				Action: func(tt *TestTUI) {
					tt.SendEscape()
				},
				Validate: func(tt *TestTUI) {
					tt.AssertModeSelection()
				},
			},
			{
				Name: "Navigate to different mode",
				Action: func(tt *TestTUI) {
					tt.SendArrowDown() // Navigate to Building
					tt.SendArrowDown() // Navigate to Debugging
					tt.SendEnter()
				},
				Validate: func(tt *TestTUI) {
					tt.AssertChatMode("🐛 Debugging")
					tt.AssertWelcomeMessage("Debugging")
				},
			},
		},
	},

	KeyboardShortcuts: Scenario{
		Name:        "Keyboard Shortcuts",
		Description: "Test all keyboard shortcuts and their functionality",
		Steps: []Step{
			{
				Name: "Initialize TUI",
				Action: func(tt *TestTUI) {
					tt.StartWithMockAgent()
				},
				Validate: func(tt *TestTUI) {
					tt.AssertModeSelection()
				},
			},
			{
				Name: "Test Ctrl+C from mode selection",
				Action: func(tt *TestTUI) {
					tt.SendCtrlC()
				},
				Validate: func(tt *TestTUI) {
					// Should quit the application
					tt.QuitWithTimeout(1 * time.Second)
				},
			},
		},
	},

	ResizeHandling: Scenario{
		Name:        "Resize Handling",
		Description: "Test TUI behavior when terminal is resized",
		Steps: []Step{
			{
				Name: "Initialize TUI with standard size",
				Action: func(tt *TestTUI) {
					tt.StartWithMockAgent()
				},
				Validate: func(tt *TestTUI) {
					tt.AssertModeSelection()
				},
			},
			{
				Name: "Simulate terminal resize",
				Action: func(tt *TestTUI) {
					// Send a window size message
					tt.SendMessage(tea.WindowSizeMsg{Width: 120, Height: 40})
				},
				Validate: func(tt *TestTUI) {
					// TUI should still be functional after resize
					tt.AssertModeSelection()
				},
			},
		},
	},
}

// CreateCustomScenario creates a custom test scenario
func CreateCustomScenario(name, description string, steps []Step) Scenario {
	return Scenario{
		Name:        name,
		Description: description,
		Steps:       steps,
	}
}

// CreateStepSequence creates a sequence of steps for complex interactions
func CreateStepSequence(actions []func(*TestTUI), validations []func(*TestTUI)) []Step {
	steps := make([]Step, 0, len(actions))
	
	for i, action := range actions {
		step := Step{
			Name:   fmt.Sprintf("Step %d", i+1),
			Action: action,
		}
		
		if i < len(validations) && validations[i] != nil {
			step.Validate = validations[i]
		}
		
		steps = append(steps, step)
	}
	
	return steps
}

// WorkflowScenarios contains end-to-end workflow scenarios
var WorkflowScenarios = struct {
	FullPlanningWorkflow Scenario
	FullBuildingWorkflow Scenario
	FullDebuggingWorkflow Scenario
	FullEnhanceWorkflow Scenario
	ModeSwithingWorkflow Scenario
}{
	FullPlanningWorkflow: Scenario{
		Name:        "Full Planning Workflow",
		Description: "Complete planning workflow from start to finish",
		Steps: []Step{
			{
				Name: "Start TUI and enter Planning mode",
				Action: func(tt *TestTUI) {
					tt.StartWithMockAgent()
					tt.SendEnter() // Select Planning mode
				},
				Validate: func(tt *TestTUI) {
					tt.AssertChatMode("📋 Planning")
					tt.AssertWelcomeMessage("Planning")
				},
			},
			{
				Name: "Request system architecture",
				Action: func(tt *TestTUI) {
					tt.MockAgent().On("Chat", mock.Anything, "Design a microservices architecture").
						Return("I'll help you design a microservices architecture. Here's a comprehensive plan...", nil)
					
					tt.SendString("Design a microservices architecture")
					tt.SendEnter()
				},
				Validate: func(tt *TestTUI) {
					tt.WaitForOutput("I'll help you design a microservices architecture", 5*time.Second)
					tt.AssertUserMessage("Design a microservices architecture")
					tt.AssertAIMessage("I'll help you design a microservices architecture")
				},
			},
			{
				Name: "Follow up with specific questions",
				Action: func(tt *TestTUI) {
					tt.MockAgent().On("Chat", mock.Anything, "What about database design?").
						Return("For database design in microservices, I recommend...", nil)
					
					tt.SendString("What about database design?")
					tt.SendEnter()
				},
				Validate: func(tt *TestTUI) {
					tt.WaitForOutput("For database design in microservices", 5*time.Second)
					tt.AssertUserMessage("What about database design?")
					tt.AssertAIMessage("For database design in microservices")
				},
			},
		},
	},

	FullBuildingWorkflow: Scenario{
		Name:        "Full Building Workflow",
		Description: "Complete building workflow from start to finish",
		Steps: []Step{
			{
				Name: "Start TUI and enter Building mode",
				Action: func(tt *TestTUI) {
					tt.StartWithMockAgent()
					tt.SendArrowDown() // Navigate to Building
					tt.SendEnter()
				},
				Validate: func(tt *TestTUI) {
					tt.AssertChatMode("🔨 Building")
					tt.AssertWelcomeMessage("Building")
				},
			},
			{
				Name: "Request code generation",
				Action: func(tt *TestTUI) {
					tt.MockAgent().On("Chat", mock.Anything, "Create a REST API handler").
						Return("Here's a REST API handler:\n\nfunc Handler(w http.ResponseWriter, r *http.Request) {\n  // implementation\n}", nil)
					
					tt.SendString("Create a REST API handler")
					tt.SendEnter()
				},
				Validate: func(tt *TestTUI) {
					tt.WaitForOutput("Here's a REST API handler", 5*time.Second)
					tt.AssertUserMessage("Create a REST API handler")
					tt.AssertAIMessage("Here's a REST API handler")
				},
			},
		},
	},

	FullDebuggingWorkflow: Scenario{
		Name:        "Full Debugging Workflow",
		Description: "Complete debugging workflow from start to finish",
		Steps: []Step{
			{
				Name: "Start TUI and enter Debugging mode",
				Action: func(tt *TestTUI) {
					tt.StartWithMockAgent()
					tt.SendArrowDown() // Navigate to Building
					tt.SendArrowDown() // Navigate to Debugging
					tt.SendEnter()
				},
				Validate: func(tt *TestTUI) {
					tt.AssertChatMode("🐛 Debugging")
					tt.AssertWelcomeMessage("Debugging")
				},
			},
			{
				Name: "Submit error for debugging",
				Action: func(tt *TestTUI) {
					tt.MockAgent().On("Chat", mock.Anything, "Debug: null pointer exception").
						Return("This null pointer exception is likely caused by...", nil)
					
					tt.SendString("Debug: null pointer exception")
					tt.SendEnter()
				},
				Validate: func(tt *TestTUI) {
					tt.WaitForOutput("This null pointer exception is likely caused by", 5*time.Second)
					tt.AssertUserMessage("Debug: null pointer exception")
					tt.AssertAIMessage("This null pointer exception is likely caused by")
				},
			},
		},
	},

	FullEnhanceWorkflow: Scenario{
		Name:        "Full Enhance Workflow",
		Description: "Complete enhance workflow from start to finish",
		Steps: []Step{
			{
				Name: "Start TUI and enter Enhance mode",
				Action: func(tt *TestTUI) {
					tt.StartWithMockAgent()
					tt.SendArrowDown() // Navigate to Building
					tt.SendArrowDown() // Navigate to Debugging
					tt.SendArrowDown() // Navigate to Enhance
					tt.SendEnter()
				},
				Validate: func(tt *TestTUI) {
					tt.AssertChatMode("🔧 Enhance")
					tt.AssertWelcomeMessage("Enhance")
				},
			},
			{
				Name: "Request code enhancement",
				Action: func(tt *TestTUI) {
					tt.MockAgent().On("Chat", mock.Anything, "Improve this legacy code").
						Return("Here are several improvements for your legacy code...", nil)
					
					tt.SendString("Improve this legacy code")
					tt.SendEnter()
				},
				Validate: func(tt *TestTUI) {
					tt.WaitForOutput("Here are several improvements for your legacy code", 5*time.Second)
					tt.AssertUserMessage("Improve this legacy code")
					tt.AssertAIMessage("Here are several improvements for your legacy code")
				},
			},
		},
	},

	ModeSwithingWorkflow: Scenario{
		Name:        "Mode Switching Workflow",
		Description: "Test switching between different modes during work",
		Steps: []Step{
			{
				Name: "Start in Planning mode",
				Action: func(tt *TestTUI) {
					tt.StartWithMockAgent()
					tt.SendEnter() // Select Planning mode
				},
				Validate: func(tt *TestTUI) {
					tt.AssertChatMode("📋 Planning")
				},
			},
			{
				Name: "Do some planning work",
				Action: func(tt *TestTUI) {
					tt.MockAgent().On("Chat", mock.Anything, "Plan a feature").
						Return("Here's a plan for your feature...", nil)
					
					tt.SendString("Plan a feature")
					tt.SendEnter()
				},
				Validate: func(tt *TestTUI) {
					tt.WaitForOutput("Here's a plan for your feature", 5*time.Second)
				},
			},
			{
				Name: "Switch to Building mode",
				Action: func(tt *TestTUI) {
					tt.SendEscape() // Return to mode selection
					tt.SendArrowDown() // Navigate to Building
					tt.SendEnter()
				},
				Validate: func(tt *TestTUI) {
					tt.AssertChatMode("🔨 Building")
				},
			},
			{
				Name: "Do building work",
				Action: func(tt *TestTUI) {
					tt.MockAgent().On("Chat", mock.Anything, "Implement the feature").
						Return("Here's the implementation...", nil)
					
					tt.SendString("Implement the feature")
					tt.SendEnter()
				},
				Validate: func(tt *TestTUI) {
					tt.WaitForOutput("Here's the implementation", 5*time.Second)
				},
			},
			{
				Name: "Switch to Debugging mode",
				Action: func(tt *TestTUI) {
					tt.SendEscape() // Return to mode selection
					tt.SendArrowDown() // Navigate to Debugging
					tt.SendEnter()
				},
				Validate: func(tt *TestTUI) {
					tt.AssertChatMode("🐛 Debugging")
				},
			},
		},
	},
}

