package protocol

import (
	"encoding/json"
	"fmt"
)

// ProtocolVersion is the current wire protocol version.
const ProtocolVersion = "1.0"

// SupportedVersions lists protocol versions this library understands.
var SupportedVersions = []string{ProtocolVersion}

// MessageType represents the high level category of a message.
type MessageType string

const (
	// MessageTypeRequest is used for client requests.
	MessageTypeRequest MessageType = "request"
	// MessageTypeResponse is used for server responses.
	MessageTypeResponse MessageType = "response"
	// MessageTypeEvent is used for asynchronous events.
	MessageTypeEvent MessageType = "event"
	// MessageTypeError is used to signal an error occurred.
	MessageTypeError MessageType = "error"
	// MessageTypeVersion is used during the initial handshake.
	MessageTypeVersion MessageType = "version"
)

// ErrorCode identifies a specific error condition.
type ErrorCode string

const (
	// ErrInvalidRequest indicates the request was malformed.
	ErrInvalidRequest ErrorCode = "invalid_request"
	// ErrInternal is returned for unexpected server errors.
	ErrInternal ErrorCode = "internal_error"
	// ErrUnsupportedVersion is returned when the peer uses
	// an unsupported protocol version.
	ErrUnsupportedVersion ErrorCode = "unsupported_version"
)

// Error represents a wire protocol error.
type Error struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Envelope wraps all protocol messages.
type Envelope struct {
	Type    MessageType     `json:"type"`
	ID      string          `json:"id,omitempty"`
	Version string          `json:"version,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// Event represents a streaming event payload.
type Event struct {
	Name string      `json:"name"`
	Data interface{} `json:"data,omitempty"`
}

// StreamChunk represents a piece of streamed text data.
type StreamChunk struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Done    bool   `json:"done"`
}

// NegotiateVersion checks if the peerVersion is supported and
// returns the version to use or an error if not supported.
func NegotiateVersion(peerVersion string) (string, error) {
	for _, v := range SupportedVersions {
		if v == peerVersion {
			return v, nil
		}
	}
	return "", &Error{Code: ErrUnsupportedVersion, Message: fmt.Sprintf("version %s not supported", peerVersion)}
}
