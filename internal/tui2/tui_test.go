package tui2

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/hammie/rubrduck/internal/config"
	tuiTesting "github.com/hammie/rubrduck/internal/tui2/testing"
)

// TUITestSuite is the main test suite for TUI functionality
type TUITestSuite struct {
	suite.Suite
	testTUI *tuiTesting.TestTUI
}

// SetupTest runs before each test
func (suite *TUITestSuite) SetupTest() {
	// Set consistent color profile for testing
	lipgloss.SetColorProfile(termenv.ANSI)
	
	// Create a new test TUI instance for each test
	suite.testTUI = tuiTesting.NewTestTUI(suite.T(),
		tuiTesting.WithSize(80, 24),
		tuiTesting.WithConfig(&config.Config{
			Model: "test-model",
		}),
	)
}

// TearDownTest runs after each test
func (suite *TUITestSuite) TearDownTest() {
	if suite.testTUI != nil {
		suite.testTUI.Quit()
	}
}

// TestInitialDisplay tests the initial display of the TUI
func (suite *TUITestSuite) TestInitialDisplay() {
	suite.testTUI.RunScenario(tuiTesting.PredefinedScenarios.InitialDisplay)
}

// TestModeNavigation tests navigation between modes
func (suite *TUITestSuite) TestModeNavigation() {
	suite.testTUI.RunScenario(tuiTesting.PredefinedScenarios.ModeNavigation)
}

// TestModeSelection tests mode selection functionality
func (suite *TUITestSuite) TestModeSelection() {
	suite.testTUI.RunScenario(tuiTesting.PredefinedScenarios.ModeSelection)
}

// TestChatInteraction tests chat interaction with AI
func (suite *TUITestSuite) TestChatInteraction() {
	suite.testTUI.RunScenario(tuiTesting.PredefinedScenarios.ChatInteraction)
}

// TestAIResponseHandling tests AI response handling
func (suite *TUITestSuite) TestAIResponseHandling() {
	suite.testTUI.RunScenario(tuiTesting.PredefinedScenarios.AIResponseHandling)
}

// TestErrorHandling tests error handling scenarios
func (suite *TUITestSuite) TestErrorHandling() {
	suite.testTUI.RunScenario(tuiTesting.PredefinedScenarios.ErrorHandling)
}

// TestNavigationBackToModes tests navigation back to mode selection
func (suite *TUITestSuite) TestNavigationBackToModes() {
	suite.testTUI.RunScenario(tuiTesting.PredefinedScenarios.NavigationBackToModes)
}

// TestKeyboardShortcuts tests keyboard shortcuts
func (suite *TUITestSuite) TestKeyboardShortcuts() {
	suite.testTUI.RunScenario(tuiTesting.PredefinedScenarios.KeyboardShortcuts)
}

// TestResizeHandling tests resize handling
func (suite *TUITestSuite) TestResizeHandling() {
	suite.testTUI.RunScenario(tuiTesting.PredefinedScenarios.ResizeHandling)
}

// TestFullPlanningWorkflow tests the complete planning workflow
func (suite *TUITestSuite) TestFullPlanningWorkflow() {
	suite.testTUI.RunScenario(tuiTesting.WorkflowScenarios.FullPlanningWorkflow)
}

// TestFullBuildingWorkflow tests the complete building workflow
func (suite *TUITestSuite) TestFullBuildingWorkflow() {
	suite.testTUI.RunScenario(tuiTesting.WorkflowScenarios.FullBuildingWorkflow)
}

// TestFullDebuggingWorkflow tests the complete debugging workflow
func (suite *TUITestSuite) TestFullDebuggingWorkflow() {
	suite.testTUI.RunScenario(tuiTesting.WorkflowScenarios.FullDebuggingWorkflow)
}

// TestFullEnhanceWorkflow tests the complete enhance workflow
func (suite *TUITestSuite) TestFullEnhanceWorkflow() {
	suite.testTUI.RunScenario(tuiTesting.WorkflowScenarios.FullEnhanceWorkflow)
}

// TestModeSwitchingWorkflow tests mode switching workflow
func (suite *TUITestSuite) TestModeSwitchingWorkflow() {
	suite.testTUI.RunScenario(tuiTesting.WorkflowScenarios.ModeSwithingWorkflow)
}

// TestCustomScenario demonstrates how to create custom test scenarios
func (suite *TUITestSuite) TestCustomScenario() {
	customScenario := tuiTesting.CreateCustomScenario(
		"Custom Planning Test",
		"A custom test for planning functionality",
		[]tuiTesting.Step{
			{
				Name: "Initialize and select Planning mode",
				Action: func(tt *tuiTesting.TestTUI) {
					tt.StartWithMockAgent()
					tt.SendEnter()
				},
				Validate: func(tt *tuiTesting.TestTUI) {
					tt.AssertChatMode("📋 Planning")
					tt.AssertWelcomeMessage("Planning")
				},
			},
			{
				Name: "Test custom planning interaction",
				Action: func(tt *tuiTesting.TestTUI) {
					tt.MockAgent().On("Chat", mock.Anything, "Plan a mobile app").
						Return("Here's a comprehensive plan for your mobile app...", nil)
					
					tt.SendString("Plan a mobile app")
					tt.SendEnter()
				},
				Validate: func(tt *tuiTesting.TestTUI) {
					tt.WaitForOutput("Here's a comprehensive plan for your mobile app", 5*time.Second)
					tt.AssertUserMessage("Plan a mobile app")
					tt.AssertAIMessage("Here's a comprehensive plan for your mobile app")
				},
			},
		},
	)
	
	suite.testTUI.RunScenario(customScenario)
}

// TestManualInteractions demonstrates manual test interactions
func (suite *TUITestSuite) TestManualInteractions() {
	// Start the TUI
	suite.testTUI.StartWithMockAgent()
	
	// Verify initial state
	suite.testTUI.AssertModeSelection()
	suite.testTUI.AssertModeSelected(0)
	
	// Navigate to Building mode
	suite.testTUI.SendArrowDown()
	suite.testTUI.AssertModeSelected(1)
	
	// Select Building mode
	suite.testTUI.SendEnter()
	suite.testTUI.AssertChatMode("🔨 Building")
	
	// Set up mock response
	suite.testTUI.MockAgent().On("Chat", mock.Anything, "Create a web server").
		Return("Here's a simple web server implementation...", nil)
	
	// Send a message
	suite.testTUI.SendString("Create a web server")
	suite.testTUI.SendEnter()
	
	// Verify the message was sent
	suite.testTUI.AssertUserMessage("Create a web server")
	suite.testTUI.AssertLoadingIndicator()
	
	// Wait for response
	suite.testTUI.WaitForOutput("Here's a simple web server implementation", 5*time.Second)
	suite.testTUI.AssertAIMessage("Here's a simple web server implementation")
	
	// Return to mode selection
	suite.testTUI.SendEscape()
	suite.testTUI.AssertModeSelection()
}

// TestAdvancedInteractions demonstrates advanced testing features
func (suite *TUITestSuite) TestAdvancedInteractions() {
	// Start the TUI
	suite.testTUI.StartWithMockAgent()
	
	// Test multiple mode switches
	for i := 0; i < 4; i++ {
		suite.testTUI.SendArrowDown()
		suite.testTUI.AssertModeSelected((i + 1) % 4)
	}
	
	// Test entering and exiting modes
	suite.testTUI.SendEnter() // Enter Planning mode
	suite.testTUI.AssertChatMode("📋 Planning")
	
	suite.testTUI.SendEscape() // Return to selection
	suite.testTUI.AssertModeSelection()
	
	// Test scrolling in chat mode
	suite.testTUI.SendArrowDown() // Navigate to Building
	suite.testTUI.SendEnter()
	suite.testTUI.AssertChatMode("🔨 Building")
	
	// Add multiple messages to test scrolling
	for i := 0; i < 5; i++ {
		message := fmt.Sprintf("Message %d", i+1)
		response := fmt.Sprintf("Response to message %d", i+1)
		
		suite.testTUI.MockAgent().On("Chat", mock.Anything, message).
			Return(response, nil)
		
		suite.testTUI.SendString(message)
		suite.testTUI.SendEnter()
		
		suite.testTUI.WaitForOutput(response, 2*time.Second)
	}
	
	// Test scrolling
	suite.testTUI.SendArrowUp() // Scroll up
	suite.testTUI.SendArrowDown() // Scroll down
}

// TestErrorScenarios demonstrates error testing
func (suite *TUITestSuite) TestErrorScenarios() {
	// Start the TUI and enter Debugging mode
	suite.testTUI.StartWithMockAgent()
	suite.testTUI.SendArrowDown() // Navigate to Building
	suite.testTUI.SendArrowDown() // Navigate to Debugging
	suite.testTUI.SendEnter()
	
	// Test timeout error
	suite.testTUI.MockAgent().On("Chat", mock.Anything, "Slow request").
		Return("", context.DeadlineExceeded)
	
	suite.testTUI.SendString("Slow request")
	suite.testTUI.SendEnter()
	
	suite.testTUI.WaitForOutput("❌ Error:", 5*time.Second)
	suite.testTUI.AssertOutput("context deadline exceeded")
	
	// Test network error
	suite.testTUI.MockAgent().On("Chat", mock.Anything, "Network request").
		Return("", errors.New("network error"))
	
	suite.testTUI.SendString("Network request")
	suite.testTUI.SendEnter()
	
	suite.testTUI.WaitForOutput("❌ Error:", 5*time.Second)
	suite.testTUI.AssertOutput("network error")
}

// TestSpecialCharacters tests handling of special characters
func (suite *TUITestSuite) TestSpecialCharacters() {
	suite.testTUI.StartWithMockAgent()
	suite.testTUI.SendEnter() // Enter Planning mode
	
	// Test special characters in input
	specialMessage := "Test with special chars: 🚀 & < > \" ' \\ / @#$%^&*()"
	suite.testTUI.MockAgent().On("Chat", mock.Anything, specialMessage).
		Return("I can handle special characters!", nil)
	
	suite.testTUI.SendString(specialMessage)
	suite.testTUI.SendEnter()
	
	suite.testTUI.WaitForOutput("I can handle special characters!", 5*time.Second)
	suite.testTUI.AssertUserMessage(specialMessage)
}

// TestLongMessages tests handling of long messages
func (suite *TUITestSuite) TestLongMessages() {
	suite.testTUI.StartWithMockAgent()
	suite.testTUI.SendEnter() // Enter Planning mode
	
	// Test long message
	longMessage := strings.Repeat("This is a very long message that should test the wrapping capabilities of the TUI. ", 10)
	suite.testTUI.MockAgent().On("Chat", mock.Anything, longMessage).
		Return("I received your long message and here's an equally long response: "+longMessage, nil)
	
	suite.testTUI.SendString(longMessage)
	suite.testTUI.SendEnter()
	
	suite.testTUI.WaitForOutput("I received your long message", 5*time.Second)
	suite.testTUI.AssertUserMessage(longMessage)
}

// TestPerformance tests performance aspects
func (suite *TUITestSuite) TestPerformance() {
	start := time.Now()
	
	// Test rapid key presses
	suite.testTUI.StartWithMockAgent()
	
	for i := 0; i < 10; i++ {
		suite.testTUI.SendArrowDown()
		suite.testTUI.SendArrowUp()
	}
	
	duration := time.Since(start)
	suite.T().Logf("Rapid key presses took: %v", duration)
	
	// Ensure it's still responsive
	suite.testTUI.AssertModeSelection()
}

// Run the test suite
func TestTUITestSuite(t *testing.T) {
	suite.Run(t, new(TUITestSuite))
}

// Example of how to write a simple failing test for TDD
func TestTUIFeatureNotImplemented(t *testing.T) {
	// This is an example of a failing test you might write
	// before implementing a new feature
	
	testTUI := tuiTesting.NewTestTUI(t)
	testTUI.StartWithMockAgent()
	
	// Test a feature that doesn't exist yet
	// testTUI.SendKey(tuiTesting.KeyRune('h')) // Help key
	// testTUI.AssertOutput("Help Menu")
	
	// This test would fail until you implement the help feature
	t.Skip("Feature not implemented yet - this is a placeholder for TDD")
}

// Example of testing with different terminal sizes
func TestTUIResponsiveDesign(t *testing.T) {
	testCases := []struct {
		name   string
		width  int
		height int
	}{
		{"Small Terminal", 40, 12},
		{"Standard Terminal", 80, 24},
		{"Large Terminal", 120, 40},
		{"Wide Terminal", 200, 24},
		{"Tall Terminal", 80, 60},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testTUI := tuiTesting.NewTestTUI(t, tuiTesting.WithSize(tc.width, tc.height))
			testTUI.StartWithMockAgent()
			
			// Test that the TUI works regardless of size
			testTUI.AssertModeSelection()
			testTUI.SendEnter()
			testTUI.AssertChatMode("📋 Planning")
			
			testTUI.Quit()
		})
	}
}

