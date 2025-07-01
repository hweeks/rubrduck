package protocol

import (
	"encoding/json"
	"testing"
)

func TestNegotiateVersion(t *testing.T) {
	v, err := NegotiateVersion(ProtocolVersion)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if v != ProtocolVersion {
		t.Fatalf("expected %s, got %s", ProtocolVersion, v)
	}

	if _, err := NegotiateVersion("2.0"); err == nil {
		t.Fatalf("expected error for unsupported version")
	}
}

func TestEnvelopeMarshal(t *testing.T) {
	ev := Event{Name: "test", Data: "hello"}
	payload, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	env := Envelope{Type: MessageTypeEvent, ID: "1", Payload: payload, Version: ProtocolVersion}
	data, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}

	var decoded Envelope
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	if decoded.Type != MessageTypeEvent || decoded.ID != "1" || decoded.Version != ProtocolVersion {
		t.Fatalf("unexpected envelope fields")
	}

	var got Event
	if err := json.Unmarshal(decoded.Payload, &got); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if got.Name != ev.Name || got.Data.(string) != ev.Data {
		t.Fatalf("payload mismatch")
	}
}

func TestErrorString(t *testing.T) {
	e := &Error{Code: ErrInvalidRequest, Message: "bad"}
	if e.Error() != "invalid_request: bad" {
		t.Fatalf("unexpected error string: %s", e.Error())
	}
}

func TestStreamChunkMarshal(t *testing.T) {
	chunk := StreamChunk{ID: "x", Content: "y", Done: true}
	data, err := json.Marshal(chunk)
	if err != nil {
		t.Fatalf("marshal chunk: %v", err)
	}
	var got StreamChunk
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal chunk: %v", err)
	}
	if got != chunk {
		t.Fatalf("unexpected chunk: %#v", got)
	}
}
