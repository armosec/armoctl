package clierr

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
)

func TestExit_BadInput(t *testing.T) {
	var stderr bytes.Buffer
	code := Render(&stderr, &Error{Code: CodeBadInput, Msg: "missing --cluster"})
	if code != ExitBadInput {
		t.Fatalf("code = %d, want %d", code, ExitBadInput)
	}
	var got map[string]string
	if err := json.Unmarshal(stderr.Bytes(), &got); err != nil {
		t.Fatalf("stderr is not JSON: %v: %q", err, stderr.String())
	}
	if got["code"] != "BAD_INPUT" {
		t.Fatalf("code field = %q, want BAD_INPUT", got["code"])
	}
	if got["error"] != "missing --cluster" {
		t.Fatalf("error field = %q", got["error"])
	}
}

func TestRender_PlainErrorIsServerCode(t *testing.T) {
	var stderr bytes.Buffer
	code := Render(&stderr, errors.New("boom"))
	if code != ExitServer {
		t.Fatalf("code = %d, want %d", code, ExitServer)
	}
}
