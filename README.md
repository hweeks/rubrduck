# RubrDuck - AI-Powered Coding Agent for IDEs

RubrDuck is a Go-based CLI tool inspired by OpenAI Codex, designed to provide a Cursor-like AI coding experience across multiple IDEs. Built with Cobra for CLI management and Bubble Tea for a beautiful TUI experience.

## 🎯 Project Vision

Bring the power of AI-assisted coding to every IDE through:

- A robust CLI with interactive TUI
- IDE extensions for seamless integration
- Real-time code analysis and generation
- Sandboxed code execution
- Multi-provider AI model support

## 🏗️ Architecture Overview

```
rubrduck/
├── cmd/                    # CLI commands (Cobra)
├── internal/
│   ├── ai/                # AI provider integrations
│   ├── tui/               # Bubble Tea TUI components
│   ├── sandbox/           # Code execution sandboxing
│   ├── agent/             # Core agent logic
│   ├── config/            # Configuration management
│   ├── api/               # API server for IDE extensions
│   └── mcp/               # Model Context Protocol support
├── pkg/
│   ├── protocol/          # Wire protocol definitions
│   ├── models/            # Data models
│   └── utils/             # Utility functions
├── extensions/
│   ├── vscode/            # VSCode extension
│   ├── jetbrains/         # IntelliJ IDEA extension
│   └── neovim/            # Neovim plugin
└── scripts/               # Build and deployment scripts
```

## 🚀 Core Features

### Phase 1: Foundation (MVP)

- [x] Project structure setup
- [ ] Basic CLI with Cobra
- [ ] Interactive TUI with Bubble Tea
- [ ] OpenAI integration
- [ ] File read/write operations
- [ ] Basic code generation

### Phase 2: Enhanced Agent

- [ ] Multi-provider support (Azure, Anthropic, Gemini, etc.)
- [ ] Code execution sandboxing
- [ ] Git integration
- [ ] Project context awareness
- [ ] Memory/conversation persistence

### Phase 3: IDE Integration

- [ ] API server for IDE communication
- [ ] VSCode extension
- [ ] Wire protocol implementation
- [ ] Real-time collaboration features
- [ ] IntelliJ IDEA extension

### Phase 4: Advanced Features

- [ ] MCP (Model Context Protocol) support
- [ ] Custom tool integration
- [ ] Plugin system
- [ ] Multi-language support
- [ ] Advanced sandboxing (Docker/Firecracker)

## 🛠️ Technology Stack

- **Language**: Go 1.21+
- **CLI Framework**: [Cobra](https://github.com/spf13/cobra)
- **TUI Framework**: [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- **Configuration**: [Viper](https://github.com/spf13/viper)
- **HTTP Client**: Standard library with retry logic
- **Logging**: [Zerolog](https://github.com/rs/zerolog)
- **Testing**: Standard Go testing + [Testify](https://github.com/stretchr/testify)

## 📦 Installation

### From Source

```bash
git clone https://github.com/hweeks/rubrduck
cd rubrduck
go build -o rubrduck cmd/rubrduck/main.go
./rubrduck
```

### Using Go Install

```bash
go install github.com/hweeks/rubrduck@latest
```

## 🔧 Configuration

RubrDuck uses a configuration file located at `~/.rubrduck/config.yaml`:

```yaml
# AI Provider Configuration
provider: openai
model: gpt-4

providers:
  openai:
    api_key: ${OPENAI_API_KEY}
    base_url: https://api.openai.com/v1

  azure:
    api_key: ${AZURE_API_KEY}
    base_url: https://your-resource.openai.azure.com
    api_version: 2024-02-15-preview

# Agent Settings
agent:
  approval_mode: suggest # suggest | auto-edit | full-auto
  sandbox_enabled: true
  max_retries: 3

# IDE Integration
api:
  enabled: true
  port: 8080
  auth_token: ${RUBRDUCK_AUTH_TOKEN}
```

## 🎮 Usage

### Interactive Mode

```bash
rubrduck
```

### Command Mode

```bash
rubrduck "explain this codebase"
rubrduck --mode full-auto "add error handling to all functions"
```

### API Server Mode (for IDE extensions)

```bash
rubrduck serve
```

## 🔌 IDE Extensions

### VSCode Extension

The VSCode extension communicates with the RubrDuck API server to provide:

- Inline code suggestions
- Chat interface
- Code actions
- Real-time collaboration

### Installation

```bash
# In the extensions/vscode directory
npm install
npm run build
# Install the .vsix file in VSCode
```

## 🛡️ Security Model

RubrDuck implements a multi-layered security approach:

> **Note for macOS users:**
> The built-in sandboxing uses Apple's `sandbox-exec`, which is unreliable for general-purpose command sandboxing on modern macOS. Even with permissive profiles, many commands may fail with exit status 65 due to SIP and system restrictions. This is a known limitation of the platform and not a bug in RubrDuck. For full sandbox support, use Linux.

1. **Approval Modes**

   - `suggest`: All actions require user approval
   - `auto-edit`: File edits are automatic, commands need approval
   - `full-auto`: Fully autonomous (sandboxed)

2. **Sandboxing**

   - Linux: Landlock + seccomp
   - macOS: Apple Sandbox
   - Windows: AppContainer (planned)

3. **Network Isolation**
   - Configurable network policies
   - API endpoint whitelisting

## 🧪 Development

### Prerequisites

- Go 1.21+
- Node.js 18+ (for extensions)
- Docker (optional, for sandboxing)

### Building

```bash
make build
```

### Testing

```bash
make test
```

### Running with Hot Reload

```bash
make dev
```

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## 📄 License

MIT License - see [LICENSE](LICENSE) for details.

## 🙏 Acknowledgments

- Inspired by [OpenAI Codex CLI](https://github.com/openai/codex)
- Built with amazing Go libraries from the community
