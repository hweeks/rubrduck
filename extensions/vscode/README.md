# RubrDuck VSCode Extension

RubrDuck is an AI-powered coding assistant that brings intelligent code analysis, generation, and chat capabilities directly to your VSCode editor.

## Features

### 🤖 AI Chat Sidebar

- Interactive chat interface with your AI assistant
- Context-aware responses based on your current code selection
- Streaming responses for real-time interaction
- Conversation history management

### 🔍 Selection-Based Actions

- **Explain Code**: Get detailed explanations of selected code
- **Fix Code**: Automatically identify and fix issues in your code
- **Fix File**: Run RubrDuck fixes on entire files or multiple selections
- **Generate Code**: Create new code based on natural language descriptions
- **Generate Tests**: Automatically create unit tests for your functions
- **Custom Commands**: Define your own prompts for common tasks

### 🎯 Context-Aware Assistance

- Automatically includes selected code as context
- Understands your current file and programming language
- Provides relevant suggestions and fixes
- Inline CodeLens actions for quick access

## Getting Started

1. **Install the Extension**: Install RubrDuck from the VSCode marketplace
2. **Configure Server**: Set your RubrDuck server URL in settings (default: `http://localhost:8080`)
3. **Start Coding**: Select code and right-click to see RubrDuck options, or open the chat sidebar

## Configuration

Open VSCode settings and search for "RubrDuck" to configure:

- `rubrduck.serverUrl`: The URL of your RubrDuck server (default: `http://localhost:8080`)
- `rubrduck.authToken`: Authentication token for the RubrDuck API (if required)
- `rubrduck.autoStart`: Automatically connect to server on startup (default: `true`)
- `rubrduck.enableCodeLens`: Show inline CodeLens actions (default: `true`)
- `rubrduck.customCommands`: Array of custom commands with `name` and `prompt`

## Usage

### Chat Sidebar

1. Click the RubrDuck icon in the activity bar
2. Type your questions or requests in the chat input
3. Use "Add Selection" to include your current code selection as context

### Context Menu Actions

1. Select code in your editor
2. Right-click to open the context menu
3. Choose from available RubrDuck actions:
   - **RubrDuck: Explain Code** - Get an explanation of the selected code
   - **RubrDuck: Fix Code** - Get suggestions to fix issues in the code

### Command Palette

Use `Ctrl+Shift+P` (or `Cmd+Shift+P` on Mac) and search for:

- `RubrDuck: Open Chat` - Open the chat sidebar
- `RubrDuck: Explain Code` - Explain selected code
- `RubrDuck: Fix Code` - Fix selected code
- `RubrDuck: Generate Code` - Generate new code
- `RubrDuck: Generate Tests` - Generate tests for selected code

## Commands

| Command             | Description           | Shortcut |
| ------------------- | --------------------- | -------- |
| `rubrduck.chat`     | Open chat sidebar     | -        |
| `rubrduck.explain`  | Explain selected code | -        |
| `rubrduck.fix`      | Fix selected code     | -        |
| `rubrduck.generate` | Generate new code     | -        |
| `rubrduck.test`     | Generate tests        | -        |

## Requirements

- VSCode 1.85.0 or higher
- RubrDuck server running (see main project documentation)

## Installation

### From Marketplace (when published)

1. Open VSCode
2. Go to Extensions view (`Ctrl+Shift+X`)
3. Search for "RubrDuck"
4. Click Install

### Manual Installation (Development)

1. Clone the RubrDuck repository
2. Navigate to `extensions/vscode`
3. Run `npm install`
4. Run `npm run compile`
5. Press `F5` to launch a new VSCode window with the extension loaded

## Development

### Building

```bash
npm run compile
```

### Watching for Changes

```bash
npm run watch
```

### Packaging

```bash
npm run vscode:prepublish
```

## Support

For issues, feature requests, or questions:

- GitHub Issues: [RubrDuck Repository](https://github.com/yourusername/rubrduck)
- Documentation: See main project README

## License

This extension is part of the RubrDuck project. See the main project license for details.
