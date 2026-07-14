package docker

import (
	"errors"
	"io"
	"strings"
	"testing"
)

// TestPushStreamErrorPayloadAbortsPush verifies that a Docker push stream
// carrying a `{"error": "..."}` message is treated as a failed push even
// though the stream itself decodes cleanly and reaches EOF. This is the
// exact shape of stream Docker emits when a push is denied or a layer fails
// to upload after the daemon has already reported some progress.
func TestPushStreamErrorPayloadAbortsPush(t *testing.T) {
	stream := strings.Join([]string{
		`{"status":"Preparing","id":"layer1"}`,
		`{"status":"Pushing","id":"layer1"}`,
		`{"error":"denied: requested access to the resource is denied","errorDetail":{"message":"denied: requested access to the resource is denied"}}`,
	}, "\n")

	_, _, err := decodePushStream(strings.NewReader(stream))
	if err == nil {
		t.Fatal("expected an error for a stream carrying an error payload, got nil")
	}
	if !strings.Contains(err.Error(), "denied") {
		t.Fatalf("expected error to contain the push error message, got: %v", err)
	}
}

// TestPushStreamTruncatedAbortsPush verifies that a stream cut off
// mid-message (e.g. connection dropped) surfaces a decode error instead of
// silently being treated as a successful push.
func TestPushStreamTruncatedAbortsPush(t *testing.T) {
	// The trailing object is missing its closing brace: the decoder will
	// read a partial token, hit EOF on the underlying reader, and report
	// io.ErrUnexpectedEOF rather than a clean io.EOF.
	stream := `{"status":"Preparing","id":"layer1"}
{"status":"Pushing","id":"layer1"
`

	_, _, err := decodePushStream(strings.NewReader(stream))
	if err == nil {
		t.Fatal("expected an error for a truncated stream, got nil")
	}
	if errors.Is(err, io.EOF) {
		t.Fatalf("truncated stream should not be reported as a clean EOF, got: %v", err)
	}
}

// TestPushStreamCleanSucceeds is a control case: a well-formed stream with
// no error payload should decode without error, proving decodePushStream
// doesn't fail on ordinary, successful push output.
func TestPushStreamCleanSucceeds(t *testing.T) {
	stream := strings.Join([]string{
		`{"status":"Preparing","id":"layer1"}`,
		`{"status":"Pushing","id":"layer1"}`,
		`{"status":"Pushed","id":"layer1"}`,
		`{"status":"Digest: sha256:abcdef"}`,
	}, "\n")

	layerOrder, _, err := decodePushStream(strings.NewReader(stream))
	if err != nil {
		t.Fatalf("expected no error for a clean stream, got: %v", err)
	}
	if len(layerOrder) != 1 || layerOrder[0] != "layer1" {
		t.Fatalf("expected layerOrder to contain [layer1], got: %v", layerOrder)
	}
}
