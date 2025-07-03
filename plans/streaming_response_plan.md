# Plan: Implement Streaming Responses, Tool Call Tracking, and Conversation Persistence

## Overview
This plan outlines the steps to enhance RubrDuck's TUI and agent so that chat responses stream in real time, tool calls are surfaced to the user for approval, and conversation context is preserved across interactions. It is inspired by the streaming implementation in [codex-rs `chatwidget.rs`](https://github.com/openai/codex/blob/main/codex-rs/tui/src/chatwidget.rs).

## Goals
1. **Streaming output** – display assistant messages incrementally as they are generated.
2. **Tool call tracking** – show tool calls and allow the user to approve or deny each one.
3. **Conversation history** – keep chat history between messages so the agent has context.
4. **Approval flows** – support multiple approve/deny cycles during a single request.

## Tasks
### 1. Extend the agent API
- Add an event-based interface similar to codex-rs where `StreamChat` returns a channel of events instead of a single callback.
- Emit events for:
  - `TokenChunk` – each chunk of assistant text.
  - `ToolRequest` – when the model requests a tool call and needs approval.
  - `ToolResult` – output from executed tools.
  - `Done` – final completion with token usage statistics.

### 2. Update AI provider wrappers
- Ensure each provider implements streaming via `StreamChat`. OpenAI already supports SSE; wrap other providers similarly.
- Convert streaming data into `TokenChunk` events and forward tool call information.

### 3. Modify the TUI
- Replace `makeAIRequest` with a streaming version that listens on the event channel.
- As each `TokenChunk` arrives, append it to the viewport so the user sees the text grow line by line.
- When a `ToolRequest` event is received, display a modal/prompt similar to codex-rs’s approval widget asking the user to approve or reject.
- If approved, send the result back to the agent; otherwise cancel that tool call.
- Continue streaming until `Done`.
- Keep the message being built in the conversation history so subsequent messages include full context.

### 4. Manage conversation context
- Remove the unconditional `agent.ClearHistory()` call in each mode handler. Instead, preserve history across requests until the user explicitly resets or switches modes.
- Store mode-specific conversations in the agent or TUI so that planning, building, debugging, and enhancement sessions remain separate but persistent.

### 5. Approval system integration
- Hook the agent’s `ApprovalSystem` callback into the TUI. When a tool call arrives, the callback should send a `ToolRequest` event to the UI and wait for the user’s response.
- Support multiple sequential approvals within one request (e.g., a tool call leading to another message that requests another tool).

### 6. Testing & validation
- Unit tests for the new streaming API and event handling.
- Manual testing in the TUI to verify that streaming text appears smoothly and that approving/denying tool calls works as expected.

## References
- [`chatwidget.rs` in codex-rs](https://github.com/openai/codex/blob/main/codex-rs/tui/src/chatwidget.rs) – demonstrates channel-based event streaming and approval dialogs.
- Existing `Agent.StreamChat` and `ApprovalSystem` implementations in this repository.

