# RubrDuck Codebase Summary

**RubrDuck** is an open-source, Go-based CLI tool and agent framework designed to bring AI-assisted coding to any IDE via a unified API and TUI. It follows a multi-phase roadmap—from foundational CLI/TUI functionality to advanced IDE integrations and sandboxed, autonomous workflows.

## 1. Project Vision & Purpose

- Provide a robust, interactive CLI (built with Cobra) and TUI (built with Bubble Tea) for AI-powered code generation, debugging, and refactoring.
- Expose an API server for IDE extensions (VSCode, JetBrains, Neovim) to offer inline suggestions, chat interfaces, and code actions.
- Implement multi-provider support (OpenAI, Azure, etc.), sandboxed code execution, conversation memory, and a plugin system.

## 2. Technology Stack & Structure

- **Language:** Go 1.21+ (cmd, internal, pkg directories)
- **CLI Framework:** Cobra (`cmd/rubrduck`)
- **TUI Framework:** Bubble Tea (`internal/tui`)
- **Config:** Viper (`config.example.yaml`)
- **AI Integration:** Provider abstractions in `internal/ai`
- **Sandboxing:** Platform adapters in `internal/sandbox`
- **Approval & Tools:** Core logic in `internal/agent`
- **Extensions:** VSCode, JetBrains, Neovim under `extensions/`
- **Auxiliary:** `examples/`, `plans/`, `scripts/`, `bin/` (prebuilt binary)

## 3. Core Agent Concept

The **Agent** component in `internal/agent` orchestrates conversation flows and actions:

1. **Conversation History:** Accumulates user and assistant messages (`ai.Message`).
2. **Tool Definitions:** Registers domain-specific tools (e.g., git, file I/O).
3. **Approval System:** Configurable modes (`suggest`, `auto-edit`, `full-auto`) govern when operations (tool calls or shell commands) require user consent.
4. **Streaming & Events:** Supports token-level streaming (`StreamChat`) and rich event streams (`StreamEvents`) with events for tokens, tool requests, execution begin/end, and results.
5. **Execution & Feedback:** After approval, tools execute in a sandbox; results feed back into the chat loop for further reasoning.

## 4. Current State & Next Steps

- **Scaffold Completed:** Project structure, CLI entrypoint, config loading, and provider interfaces are in place.
 - **Phase 1 (MVP) Completed:** Core CLI commands, TUI workflows, AI provider integration, file operations, and basic code generation are functional.
- **Phase 2 & Beyond:** Streaming responses, conversation persistence, IDE extensions, MCP support, advanced sandboxing are outlined in `plans/streaming_response_plan.md`.
- **Guidelines:** `AGENTS.md` details best practices for file ops and incremental updates to support agent reliability and performance.

This codebase is in an early exploratory stage: architectural building blocks and interfaces exist, but behavior and workflows are pending completion. The **agent model**—with pluggable AI providers, an approval workflow, and event-driven streaming—is the central innovation driving the roadmap forward.
