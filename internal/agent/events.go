package agent

import "github.com/hammie/rubrduck/internal/ai"

// StreamEventType represents the type of streaming event returned by StreamEvents.
type StreamEventType int

const (
	// EventTokenChunk indicates a chunk of assistant text.
	EventTokenChunk StreamEventType = iota
	// EventToolRequest indicates a tool call that requires approval.
	EventToolRequest
	// EventToolResult contains the output of an executed tool.
	EventToolResult
	// EventDone signals the end of the streaming conversation and includes usage stats.
	EventDone
)

// StreamEvent is emitted by Agent.StreamEvents to report incremental progress.
type StreamEvent struct {
	Type    StreamEventType
	Token   string
	Request *ApprovalRequest
	Result  string
	Usage   ai.Usage
	Err     error
}
