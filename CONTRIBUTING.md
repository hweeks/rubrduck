# Contributing to RubrDuck

Thank you for your interest in contributing to RubrDuck! We welcome contributions from the community and are grateful for any help you can provide.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Project Structure](#project-structure)
- [Making Contributions](#making-contributions)
- [Coding Standards](#coding-standards)
- [Testing](#testing)
- [Documentation](#documentation)
- [Submitting Changes](#submitting-changes)

## Code of Conduct

By participating in this project, you agree to abide by our Code of Conduct. Please be respectful and constructive in all interactions.

## Getting Started

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/rubrduck.git
   cd rubrduck
   ```
3. **Add the upstream remote**:
   ```bash
   git remote add upstream https://github.com/hammie/rubrduck.git
   ```

## Development Setup

### Prerequisites

- Go 1.21 or higher
- Make
- Git
- Node.js 18+ (for IDE extensions)

### Initial Setup

1. **Install dependencies**:

   ```bash
   make deps
   ```

2. **Build the project**:

   ```bash
   make build
   ```

3. **Run tests**:
   ```bash
   make test
   ```

### Development Workflow

1. **Create a feature branch**:

   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** following our coding standards

3. **Test your changes**:

   ```bash
   make test
   make lint
   ```

4. **Commit your changes** with descriptive commit messages

## Project Structure

```
rubrduck/
├── cmd/rubrduck/          # CLI entry point and commands
├── internal/              # Private application code
│   ├── ai/               # AI provider interfaces and implementations
│   ├── agent/            # Core agent logic
│   ├── api/              # API server for IDE extensions
│   ├── config/           # Configuration management
│   ├── sandbox/          # Code execution sandboxing
│   └── tui/              # Terminal UI components
├── pkg/                   # Public packages
│   ├── protocol/         # Wire protocol definitions
│   └── utils/            # Utility functions
└── extensions/            # IDE extensions
    ├── vscode/           # VSCode extension
    └── jetbrains/        # IntelliJ IDEA extension
```

## Making Contributions

### Types of Contributions

- **Bug fixes**: Fix issues reported in GitHub Issues
- **Features**: Implement new features or enhance existing ones
- **Documentation**: Improve documentation, add examples
- **Tests**: Add missing tests or improve test coverage
- **Performance**: Optimize code for better performance
- **Refactoring**: Improve code quality and maintainability

### Before You Start

1. **Check existing issues** to see if someone is already working on it
2. **Open an issue** to discuss major changes before implementing
3. **Keep changes focused** - one feature/fix per pull request

## Coding Standards

### Go Code Style

- Follow standard Go conventions and idioms
- Use `gofmt` to format your code
- Run `golangci-lint` before submitting
- Write clear, self-documenting code
- Add comments for exported functions and types

### Commit Messages

Follow the conventional commits specification:

```
type(scope): subject

body

footer
```

Types:

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Test additions or modifications
- `chore`: Build process or auxiliary tool changes

Example:

```
feat(ai): add support for Anthropic Claude provider

- Implement Anthropic provider following the Provider interface
- Add configuration options for Anthropic API
- Include streaming support
```

## Testing

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run tests for a specific package
go test ./internal/agent/...
```

### Writing Tests

- Write unit tests for all new functionality
- Aim for at least 80% code coverage
- Use table-driven tests where appropriate
- Mock external dependencies
- Test edge cases and error conditions

Example test structure:

```go
func TestFunctionName(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {
            name:  "valid input",
            input: "test",
            want:  "expected",
        },
        // Add more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := FunctionName(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("FunctionName() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("FunctionName() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Documentation

- Update README.md if adding new features
- Add godoc comments to all exported types and functions
- Include examples in documentation where helpful
- Update configuration examples if adding new options

## Submitting Changes

### Pull Request Process

1. **Update your fork**:

   ```bash
   git fetch upstream
   git checkout main
   git merge upstream/main
   ```

2. **Rebase your feature branch**:

   ```bash
   git checkout feature/your-feature-name
   git rebase main
   ```

3. **Push to your fork**:

   ```bash
   git push origin feature/your-feature-name
   ```

4. **Create a Pull Request** on GitHub

### Pull Request Guidelines

- **Title**: Clear and descriptive
- **Description**: Explain what changes you made and why
- **Testing**: Describe how you tested your changes
- **Screenshots**: Include for UI changes
- **Breaking Changes**: Clearly document any breaking changes

### Review Process

1. Automated checks must pass (tests, linting, etc.)
2. At least one maintainer review required
3. Address review feedback promptly
4. Squash commits before merging if requested

## Getting Help

- **Discord**: Join our community Discord server
- **GitHub Discussions**: Ask questions and discuss ideas
- **Issues**: Report bugs or request features

## Recognition

Contributors will be recognized in:

- The project README
- Release notes
- Our contributors page

Thank you for contributing to RubrDuck! 🦆
