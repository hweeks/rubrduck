# TUI Testing Framework

A comprehensive testing framework for Terminal User Interface (TUI) applications built with Bubble Tea, based on the `teatest` approach.

## Overview

This testing framework provides a complete solution for testing TUI applications with:

- **Automated UI Testing**: Test user interactions, key presses, and UI state changes
- **Mock Integration**: Mock AI agents and external dependencies
- **Scenario-Based Testing**: Pre-defined and custom test scenarios
- **Golden File Testing**: Capture and compare UI output
- **Performance Testing**: Test responsiveness and performance characteristics
- **Cross-Platform Testing**: Test on different terminal sizes and configurations

## Features

### Core Testing Capabilities

- ✅ **Mode Selection Testing** - Test navigation between different TUI modes
- ✅ **Chat Interface Testing** - Test chat interactions and AI responses
- ✅ **Error Handling Testing** - Test error scenarios and error display
- ✅ **Keyboard Navigation Testing** - Test all keyboard shortcuts and navigation
- ✅ **Responsive Design Testing** - Test different terminal sizes
- ✅ **Performance Testing** - Test UI responsiveness under load
- ✅ **Integration Testing** - Test complete workflows end-to-end

### Testing Utilities

- **Mock Agent**: Complete mock implementation of AI agents
- **Scenario Builder**: Create custom test scenarios with steps
- **Assertion Library**: Rich set of UI-specific assertions
- **Golden Files**: Capture and compare UI output automatically
- **Test Helpers**: Utilities for common testing patterns

## Quick Start

### 1. Install Dependencies

```bash
go get github.com/charmbracelet/x/exp/teatest@latest
go get github.com/stretchr/testify@latest
```

### 2. Basic Test Example

```go
package main

import (
    "testing"
    "github.com/hammie/rubrduck/internal/tui2/testing"
)

func TestBasicTUIInteraction(t *testing.T) {
    // Create a test TUI instance
    testTUI := testing.NewTestTUI(t)
    defer testTUI.Quit()
    
    // Start the TUI with mock agent
    testTUI.StartWithMockAgent()
    
    // Test initial display
    testTUI.AssertModeSelection()
    testTUI.AssertModeSelected(0)
    
    // Navigate and select a mode
    testTUI.SendArrowDown()
    testTUI.SendEnter()
    
    // Verify mode selection
    testTUI.AssertChatMode("🔨 Building")
}
```

### 3. Using Pre-defined Scenarios

```go
func TestPredefinedScenarios(t *testing.T) {
    testTUI := testing.NewTestTUI(t)
    defer testTUI.Quit()
    
    // Run a complete workflow scenario
    testTUI.RunScenario(testing.PredefinedScenarios.FullPlanningWorkflow)
}
```

## API Reference

### TestTUI

The main testing interface that provides all testing capabilities.

```go
type TestTUI struct {
    // Core methods
    StartWithMockAgent()                    // Initialize TUI with mock agent
    SendKey(key tea.KeyMsg)                 // Send individual key press
    SendString(s string)                    // Send string as key presses
    SendEnter()                             // Send Enter key
    SendEscape()                            // Send Escape key
    SendArrowUp()                           // Send up arrow
    SendArrowDown()                         // Send down arrow
    
    // Assertions
    AssertOutput(expected string)           // Assert output contains text
    AssertModeSelection()                   // Assert mode selection screen
    AssertChatMode(mode string)             // Assert chat mode is active
    AssertUserMessage(msg string)           // Assert user message displayed
    AssertAIMessage(msg string)             // Assert AI message displayed
    AssertLoadingIndicator()                // Assert loading indicator shown
    
    // Advanced
    WaitForOutput(text string, timeout time.Duration)  // Wait for specific output
    MockAgent() *MockAgent                              // Get mock agent for setup
    RunScenario(scenario Scenario)                      // Run test scenario
}
```

### Mock Agent

Mock implementation of AI agents for testing.

```go
// Setup mock responses
testTUI.MockAgent().On("ProcessRequest", mock.Anything, "user input", "model").
    Return("AI response", nil)

// Test error scenarios
testTUI.MockAgent().On("ProcessRequest", mock.Anything, "error input", "model").
    Return("", errors.New("network error"))
```

### Scenarios

Pre-defined test scenarios for common workflows.

```go
// Available predefined scenarios
PredefinedScenarios.InitialDisplay         // Test initial UI display
PredefinedScenarios.ModeNavigation         // Test mode navigation
PredefinedScenarios.ChatInteraction        // Test chat functionality
PredefinedScenarios.ErrorHandling          // Test error scenarios
PredefinedScenarios.KeyboardShortcuts      // Test keyboard shortcuts

// Complete workflow scenarios
WorkflowScenarios.FullPlanningWorkflow     // Complete planning workflow
WorkflowScenarios.FullBuildingWorkflow     // Complete building workflow
WorkflowScenarios.FullDebuggingWorkflow    // Complete debugging workflow
WorkflowScenarios.FullEnhanceWorkflow      // Complete enhance workflow
```

### Custom Scenarios

Create custom test scenarios for specific testing needs.

```go
customScenario := testing.CreateCustomScenario(
    "Custom Test",
    "Description of custom test",
    []testing.Step{
        {
            Name: "Step 1",
            Action: func(tt *testing.TestTUI) {
                tt.StartWithMockAgent()
                tt.SendEnter()
            },
            Validate: func(tt *testing.TestTUI) {
                tt.AssertChatMode("📋 Planning")
            },
        },
        // Add more steps...
    },
)

testTUI.RunScenario(customScenario)
```

## Testing Patterns

### 1. Test-Driven Development (TDD)

Write failing tests before implementing features:

```go
func TestNewFeatureNotImplemented(t *testing.T) {
    testTUI := testing.NewTestTUI(t)
    testTUI.StartWithMockAgent()
    
    // Test feature that doesn't exist yet
    testTUI.SendKey(testing.KeyRune('h')) // Help key
    testTUI.AssertOutput("Help Menu")     // This will fail initially
}
```

### 2. Behavior-Driven Testing

Test user behavior and workflows:

```go
func TestUserWorkflow(t *testing.T) {
    testTUI := testing.NewTestTUI(t)
    testTUI.StartWithMockAgent()
    
    // Given: User is on mode selection
    testTUI.AssertModeSelection()
    
    // When: User selects Planning mode
    testTUI.SendEnter()
    
    // Then: Planning mode should be active
    testTUI.AssertChatMode("📋 Planning")
    
    // And: Welcome message should be shown
    testTUI.AssertWelcomeMessage("Planning")
}
```

### 3. Error Testing

Test error scenarios thoroughly:

```go
func TestErrorHandling(t *testing.T) {
    testTUI := testing.NewTestTUI(t)
    testTUI.StartWithMockAgent()
    
    // Setup error response
    testTUI.MockAgent().On("ProcessRequest", mock.Anything, "error", "model").
        Return("", errors.New("network timeout"))
    
    // Test error handling
    testTUI.SendEnter() // Enter mode
    testTUI.SendString("error")
    testTUI.SendEnter()
    
    // Verify error is displayed
    testTUI.WaitForOutput("❌ Error:", 5*time.Second)
    testTUI.AssertOutput("network timeout")
}
```

### 4. Performance Testing

Test UI responsiveness:

```go
func TestPerformance(t *testing.T) {
    testTUI := testing.NewTestTUI(t)
    testTUI.StartWithMockAgent()
    
    start := time.Now()
    
    // Rapid key presses
    for i := 0; i < 100; i++ {
        testTUI.SendArrowDown()
        testTUI.SendArrowUp()
    }
    
    duration := time.Since(start)
    assert.Less(t, duration, time.Second, "UI should remain responsive")
}
```

### 5. Responsive Design Testing

Test different terminal sizes:

```go
func TestResponsiveDesign(t *testing.T) {
    sizes := []struct{ width, height int }{
        {40, 12},   // Small terminal
        {80, 24},   // Standard terminal
        {120, 40},  // Large terminal
    }
    
    for _, size := range sizes {
        testTUI := testing.NewTestTUI(t, 
            testing.WithSize(size.width, size.height))
        testTUI.StartWithMockAgent()
        testTUI.AssertModeSelection()
        testTUI.Quit()
    }
}
```

## Running Tests

### Using Go Test

```bash
# Run all TUI tests
go test ./internal/tui2/...

# Run specific test
go test -run TestModeNavigation ./internal/tui2/

# Run tests with verbose output
go test -v ./internal/tui2/...

# Run tests with coverage
go test -cover ./internal/tui2/...

# Update golden files
go test -update ./internal/tui2/...
```

### Using Make

```bash
# Run all tests
make test

# Run TUI tests specifically
make test-tui

# Run tests with coverage
make test-coverage

# Update golden files
make test-update

# Run tests in CI mode
make test-ci
```

## CI/CD Integration

### GitHub Actions

```yaml
name: TUI Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - uses: actions/setup-go@v4
      with:
        go-version: '1.24'
    
    - name: Run TUI tests
      run: make test-tui
      
    - name: Check test coverage
      run: make test-coverage
```

### Color Profile Consistency

For consistent testing across different environments:

```go
func init() {
    // Set consistent color profile for testing
    lipgloss.SetColorProfile(termenv.Ascii)
}
```

## Best Practices

### 1. Test Organization

- **Group related tests**: Use test suites for organization
- **Use descriptive names**: Test names should describe what they test
- **Keep tests isolated**: Each test should be independent
- **Use setup/teardown**: Clean up resources after tests

### 2. Mock Management

- **Setup mocks properly**: Configure mock responses before actions
- **Test both success and failure**: Test happy path and error cases
- **Use realistic data**: Mock responses should be realistic
- **Verify mock calls**: Assert that mocks were called as expected

### 3. Assertions

- **Use specific assertions**: Use the most specific assertion available
- **Test user-visible behavior**: Focus on what users see and experience
- **Test edge cases**: Test boundary conditions and edge cases
- **Use timeouts**: Set appropriate timeouts for async operations

### 4. Golden Files

- **Review golden files**: Always review generated golden files
- **Keep them minimal**: Only capture necessary output
- **Version control**: Check golden files into version control
- **Update carefully**: Only update when output legitimately changes

## Troubleshooting

### Common Issues

1. **Tests failing in CI but passing locally**
   - Check color profile settings
   - Verify line endings in golden files
   - Ensure consistent terminal size

2. **Flaky tests**
   - Add appropriate timeouts
   - Use WaitForOutput for async operations
   - Check for race conditions

3. **Mock not working**
   - Verify mock setup before actions
   - Check mock argument matchers
   - Ensure mock is properly configured

### Debug Mode

Enable debug mode for detailed output:

```go
testTUI := testing.NewTestTUI(t, testing.WithDebug(true))
```

### Logging

Add logging to understand test execution:

```go
testTUI.MockAgent().On("ProcessRequest", mock.Anything, mock.Anything, mock.Anything).
    Run(func(args mock.Arguments) {
        t.Logf("Mock called with: %s", args.Get(1))
    }).
    Return("response", nil)
```

## Contributing

When adding new testing features:

1. **Add tests for your tests**: Meta-testing is important
2. **Update documentation**: Keep README up to date
3. **Add examples**: Provide usage examples
4. **Consider edge cases**: Think about unusual scenarios
5. **Maintain compatibility**: Don't break existing tests

## Advanced Features

### Custom Assertions

Create custom assertions for domain-specific testing:

```go
func (tt *TestTUI) AssertPlanningModeActive() {
    tt.AssertChatMode("📋 Planning")
    tt.AssertOutput("Planning Mode")
    tt.AssertWelcomeMessage("Planning")
}
```

### Test Data Management

Use test data files for complex scenarios:

```go
func TestWithTestData(t *testing.T) {
    testData := loadTestData("testdata/planning_scenario.json")
    testTUI := testing.NewTestTUI(t)
    // Use test data in your tests
}
```

### Parallel Testing

Run tests in parallel for faster execution:

```go
func TestParallel(t *testing.T) {
    t.Parallel()
    testTUI := testing.NewTestTUI(t)
    // Test implementation
}
```

## Examples

See `tui_test.go` for comprehensive examples of:
- Basic TUI testing
- Scenario-based testing
- Error handling testing
- Performance testing
- Responsive design testing
- Custom scenario creation

## License

This testing framework is part of the RubrDuck project and follows the same license terms.