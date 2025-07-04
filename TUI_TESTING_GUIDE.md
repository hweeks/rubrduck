# TUI Testing Guide for RubrDuck

A comprehensive guide for writing failing tests and implementing TUI features using Test-Driven Development (TDD) with the teatest framework.

## Overview

This guide provides everything you need to write failing tests for TUI features before implementing them, following TDD principles. The testing framework is built on the teatest approach and provides comprehensive tools for testing Bubble Tea applications.

## 🎯 Quick Start: Write Your First Failing Test

### 1. Install Dependencies

```bash
make deps-test
```

### 2. Write a Failing Test

Create a new test file or add to existing ones:

```go
// internal/tui2/feature_test.go
package tui2

import (
    "testing"
    "time"
    
    tuiTesting "github.com/hammie/rubrduck/internal/tui2/testing"
)

func TestHelpMenuFeature(t *testing.T) {
    // This test will fail until you implement the help menu
    testTUI := tuiTesting.NewTestTUI(t)
    defer testTUI.Quit()
    
    testTUI.StartWithMockAgent()
    
    // Test pressing 'h' key to show help
    testTUI.SendKey(tuiTesting.KeyRune('h'))
    
    // Assert help menu appears
    testTUI.WaitForOutput("Help Menu", 2*time.Second)
    testTUI.AssertOutput("Keyboard Shortcuts:")
    testTUI.AssertOutput("h - Show help")
    testTUI.AssertOutput("q - Quit")
    testTUI.AssertOutput("ESC - Back to modes")
}
```

### 3. Run the Failing Test

```bash
make tdd TEST=TestHelpMenuFeature
```

The test will fail because the help menu feature doesn't exist yet. This is expected and desired in TDD!

### 4. Implement the Feature

Now implement the minimum code to make the test pass:

```go
// In your TUI update function, add:
case tea.KeyMsg:
    if msg.Runes[0] == 'h' {
        return m.showHelpMenu()
    }
    // ... existing code
```

### 5. Run the Test Again

```bash
make tdd TEST=TestHelpMenuFeature
```

Repeat until the test passes!

## 🛠 Available Testing Tools

### Core Testing Framework

The framework provides these main components:

1. **TestTUI**: Main testing interface
2. **MockAgent**: Mock AI agent for testing
3. **Scenarios**: Pre-built test scenarios
4. **Assertions**: UI-specific assertions
5. **Golden Files**: Output comparison testing

### Key Testing Methods

```go
// Setup and control
testTUI.StartWithMockAgent()                    // Start TUI with mock
testTUI.SendString("hello")                     // Type text
testTUI.SendEnter()                             // Press Enter
testTUI.SendEscape()                            // Press Escape
testTUI.SendArrowUp()                           // Arrow keys
testTUI.SendArrowDown()

// Waiting and timing
testTUI.WaitForOutput("text", 5*time.Second)    // Wait for output
testTUI.WaitForOutputPattern("pattern", timeout) // Wait for pattern

// Assertions
testTUI.AssertOutput("expected text")           // Check output contains
testTUI.AssertOutputNotContains("unwanted")     // Check output doesn't contain
testTUI.AssertModeSelection()                   // Check mode selection screen
testTUI.AssertChatMode("Mode Name")             // Check chat mode active
testTUI.AssertUserMessage("message")            // Check user message
testTUI.AssertAIMessage("response")             // Check AI response
testTUI.AssertLoadingIndicator()                // Check loading spinner

// Mock setup
testTUI.MockAgent().On("ProcessRequest", mock.Anything, "input", "model").
    Return("response", nil)
```

## 📝 TDD Workflow Examples

### Example 1: Adding Search Functionality

**Step 1: Write failing test**

```go
func TestSearchFeature(t *testing.T) {
    testTUI := tuiTesting.NewTestTUI(t)
    defer testTUI.Quit()
    
    testTUI.StartWithMockAgent()
    testTUI.SendEnter() // Enter Planning mode
    
    // Test search shortcut
    testTUI.SendKey(tuiTesting.KeyRune('/'))
    testTUI.AssertOutput("Search:")
    
    // Test search functionality
    testTUI.SendString("test query")
    testTUI.SendEnter()
    testTUI.AssertOutput("Searching for: test query")
}
```

**Step 2: Run failing test**
```bash
make tdd TEST=TestSearchFeature
```

**Step 3: Implement minimal feature**
Add search handling to your TUI update method.

**Step 4: Iterate until passing**

### Example 2: Adding Export Functionality

**Step 1: Write failing test**

```go
func TestExportConversation(t *testing.T) {
    testTUI := tuiTesting.NewTestTUI(t)
    defer testTUI.Quit()
    
    testTUI.StartWithMockAgent()
    testTUI.SendEnter() // Enter mode
    
    // Add some conversation
    testTUI.MockAgent().On("ProcessRequest", mock.Anything, "hello", "test-model").
        Return("Hello there!", nil)
    
    testTUI.SendString("hello")
    testTUI.SendEnter()
    testTUI.WaitForOutput("Hello there!", 2*time.Second)
    
    // Test export
    testTUI.SendKey(tuiTesting.KeyRune('e'))
    testTUI.AssertOutput("Export conversation")
    testTUI.AssertOutput("Conversation exported to:")
}
```

### Example 3: Adding Theme Selection

**Step 1: Write failing test**

```go
func TestThemeSelection(t *testing.T) {
    testTUI := tuiTesting.NewTestTUI(t)
    defer testTUI.Quit()
    
    testTUI.StartWithMockAgent()
    
    // Test theme menu
    testTUI.SendKey(tuiTesting.KeyRune('t'))
    testTUI.AssertOutput("Theme Selection")
    testTUI.AssertOutput("1. Dark")
    testTUI.AssertOutput("2. Light")
    testTUI.AssertOutput("3. High Contrast")
    
    // Test theme selection
    testTUI.SendKey(tuiTesting.KeyRune('2'))
    testTUI.AssertOutput("Light theme activated")
}
```

## 🎭 Advanced Testing Scenarios

### Custom Scenario Creation

```go
func TestComplexWorkflow(t *testing.T) {
    scenario := tuiTesting.CreateCustomScenario(
        "Complex Feature Workflow",
        "Test a complex multi-step feature",
        []tuiTesting.Step{
            {
                Name: "Setup initial state",
                Action: func(tt *tuiTesting.TestTUI) {
                    tt.StartWithMockAgent()
                    tt.SendEnter() // Enter Planning mode
                },
                Validate: func(tt *tuiTesting.TestTUI) {
                    tt.AssertChatMode("📋 Planning")
                },
            },
            {
                Name: "Test new feature",
                Action: func(tt *tuiTesting.TestTUI) {
                    // Test your new feature here
                    tt.SendKey(tuiTesting.KeyRune('n')) // New feature key
                },
                Validate: func(tt *tuiTesting.TestTUI) {
                    // Assert expected behavior
                    tt.AssertOutput("New feature activated")
                },
            },
        },
    )
    
    testTUI := tuiTesting.NewTestTUI(t)
    defer testTUI.Quit()
    
    testTUI.RunScenario(scenario)
}
```

### Error Scenario Testing

```go
func TestErrorHandlingFeature(t *testing.T) {
    testTUI := tuiTesting.NewTestTUI(t)
    defer testTUI.Quit()
    
    testTUI.StartWithMockAgent()
    testTUI.SendEnter() // Enter mode
    
    // Test error handling for new feature
    testTUI.MockAgent().On("ProcessRequest", mock.Anything, "error command", "test-model").
        Return("", errors.New("feature not available"))
    
    testTUI.SendString("error command")
    testTUI.SendEnter()
    
    testTUI.WaitForOutput("❌ Error:", 5*time.Second)
    testTUI.AssertOutput("feature not available")
    testTUI.AssertOutput("Please try again or contact support")
}
```

### Performance Testing

```go
func TestNewFeaturePerformance(t *testing.T) {
    testTUI := tuiTesting.NewTestTUI(t)
    defer testTUI.Quit()
    
    testTUI.StartWithMockAgent()
    
    start := time.Now()
    
    // Test rapid feature usage
    for i := 0; i < 50; i++ {
        testTUI.SendKey(tuiTesting.KeyRune('f')) // Feature key
        testTUI.SendEscape() // Cancel
    }
    
    duration := time.Since(start)
    assert.Less(t, duration, 2*time.Second, "Feature should be responsive")
}
```

## 🧪 Testing Patterns

### 1. Red-Green-Refactor Pattern

```bash
# Red: Write failing test
make tdd TEST=TestNewFeature

# Green: Make it pass with minimal code
# (implement feature)
make tdd TEST=TestNewFeature

# Refactor: Improve the code
# (refactor while keeping tests green)
make tdd TEST=TestNewFeature
```

### 2. Outside-In Testing

Start with high-level behavior tests, then drill down:

```go
// High-level test
func TestUserCanCreateProject(t *testing.T) {
    // Test complete user workflow
}

// Lower-level tests
func TestProjectCreationForm(t *testing.T) {
    // Test form validation
}

func TestProjectSaving(t *testing.T) {
    // Test save functionality
}
```

### 3. Triangulation

Write multiple tests that drive you toward a general solution:

```go
func TestHelpWithSingleCommand(t *testing.T) {
    // Test help with one command
}

func TestHelpWithMultipleCommands(t *testing.T) {
    // Test help with many commands
}

func TestHelpWithNoCommands(t *testing.T) {
    // Test edge case
}
```

## 🎯 Feature Ideas to Test-Drive

Here are some feature ideas you can implement using TDD:

### Basic Features
- [ ] Help menu (press 'h')
- [ ] Search conversations (press '/')
- [ ] Export conversation (press 'e')
- [ ] Theme selection (press 't')
- [ ] Font size adjustment ('+'/'-')
- [ ] Conversation history (press 'H')
- [ ] Bookmarks (press 'b')
- [ ] Quick actions menu (press 'q')

### Advanced Features
- [ ] Split screen mode
- [ ] Multiple conversation tabs
- [ ] Custom keyboard shortcuts
- [ ] Plugin system
- [ ] Conversation templates
- [ ] Auto-save functionality
- [ ] Conversation sharing
- [ ] Voice input mode

### Integration Features
- [ ] File attachment support
- [ ] Code execution environment
- [ ] External tool integration
- [ ] API endpoint testing
- [ ] Database query interface
- [ ] Git integration
- [ ] CI/CD pipeline integration

## 📊 Running Tests

### Development Workflow

```bash
# Run all TUI tests
make test-tui

# Run specific test during development
make tdd TEST=TestYourFeature

# Run tests with coverage
make test-tui-coverage

# Update golden files when UI changes
make test-update

# Run failing tests (for TDD)
make test-failing

# Run predefined scenarios
make test-scenarios

# Clean up test artifacts
make test-clean
```

### CI/CD Integration

```bash
# Run tests in CI mode
make test-ci

# Run only TUI tests in CI
make test-tui-ci

# Run performance tests
make test-performance
```

## 🔧 Debugging Tests

### Common Issues and Solutions

1. **Test timing issues**
   ```go
   // Use appropriate timeouts
   testTUI.WaitForOutput("expected", 5*time.Second)
   ```

2. **Mock setup problems**
   ```go
   // Setup mocks before actions
   testTUI.MockAgent().On("ProcessRequest", mock.Anything, "input", "model").
       Return("response", nil)
   ```

3. **UI state issues**
   ```go
   // Check intermediate states
   testTUI.AssertModeSelection()
   testTUI.SendEnter()
   testTUI.AssertChatMode("Expected Mode")
   ```

### Debug Mode

```go
// Add debug logging
testTUI.MockAgent().On("ProcessRequest", mock.Anything, mock.Anything, mock.Anything).
    Run(func(args mock.Arguments) {
        t.Logf("Mock called with: %s", args.Get(1))
    }).
    Return("response", nil)
```

## 📚 Best Practices

### Writing Good Failing Tests

1. **Test behavior, not implementation**
   ```go
   // Good: Test what user sees
   testTUI.AssertOutput("Welcome message")
   
   // Bad: Test internal state
   // assert.Equal(t, "welcome", model.message)
   ```

2. **Use descriptive test names**
   ```go
   // Good
   func TestUserCanExportConversationToFile(t *testing.T)
   
   // Bad
   func TestExport(t *testing.T)
   ```

3. **Test one thing at a time**
   ```go
   // Good: Focused test
   func TestHelpMenuDisplaysShortcuts(t *testing.T)
   func TestHelpMenuCanBeClosed(t *testing.T)
   
   // Bad: Testing too much
   func TestHelpMenuEverything(t *testing.T)
   ```

4. **Make tests readable**
   ```go
   // Use descriptive helper methods
   func (tt *TestTUI) EnterPlanningMode() {
       tt.SendEnter()
       tt.AssertChatMode("📋 Planning")
   }
   ```

### Test Organization

1. **Group related tests**
   ```go
   // Use test suites for organization
   type HelpMenuTestSuite struct {
       suite.Suite
       testTUI *tuiTesting.TestTUI
   }
   ```

2. **Use setup and teardown**
   ```go
   func (suite *HelpMenuTestSuite) SetupTest() {
       suite.testTUI = tuiTesting.NewTestTUI(suite.T())
   }
   
   func (suite *HelpMenuTestSuite) TearDownTest() {
       suite.testTUI.Quit()
   }
   ```

### Mock Management

1. **Set up mocks before actions**
2. **Use realistic test data**
3. **Test both success and error cases**
4. **Verify mock expectations**

## 🚀 Getting Started Checklist

- [ ] Install dependencies: `make deps-test`
- [ ] Read the framework documentation: `internal/tui2/testing/README.md`
- [ ] Run existing tests: `make test-tui`
- [ ] Write your first failing test
- [ ] Run the failing test: `make tdd TEST=YourTest`
- [ ] Implement minimal code to pass
- [ ] Refactor and improve
- [ ] Add more test cases
- [ ] Update documentation

## 📖 Additional Resources

- [Bubble Tea Documentation](https://github.com/charmbracelet/bubbletea)
- [Teatest Documentation](https://carlosbecker.com/posts/teatest/)
- [Testing Framework README](internal/tui2/testing/README.md)
- [Example Tests](internal/tui2/tui_test.go)

## 🤝 Contributing

When adding new features:

1. **Write failing tests first** (TDD)
2. **Update documentation**
3. **Add examples**
4. **Test edge cases**
5. **Consider performance**
6. **Ensure CI passes**

---

**Happy Test-Driven Development!** 🎉

Remember: The goal is to write failing tests that describe the behavior you want, then implement just enough code to make them pass. This approach leads to better design, fewer bugs, and more maintainable code.