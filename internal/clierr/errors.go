// Package clierr defines typed CLI errors, exit codes, and the
// stderr-JSON error writer used by every armoctl subcommand.
package clierr

import (
	"encoding/json"
	"errors"
	"io"
)

const (
	ExitOK           = 0
	ExitBadInput     = 2
	ExitAuth         = 3
	ExitNotFound     = 4
	ExitServer       = 5
	ExitNeedsConfirm = 6
	ExitConflict     = 7
)

type Code string

const (
	CodeBadInput     Code = "BAD_INPUT"
	CodeAuth         Code = "AUTH"
	CodeNotFound     Code = "NOT_FOUND"
	CodeServer       Code = "SERVER"
	CodeNeedsConfirm Code = "NEEDS_CONFIRM"
	CodeConflict     Code = "CONFLICT"
)

// Error is an armoctl-typed error.
type Error struct {
	Code      Code   `json:"code"`
	Msg       string `json:"error"`
	Hint      string `json:"hint,omitempty"`
	RequestID string `json:"requestId,omitempty"`
}

func (e *Error) Error() string { return e.Msg }

// Render writes the error as JSON to w and returns the exit code.
// Any non-*Error value is treated as ExitServer / CodeServer.
func Render(w io.Writer, err error) int {
	var e *Error
	if !errors.As(err, &e) {
		e = &Error{Code: CodeServer, Msg: err.Error()}
	}
	b, _ := json.Marshal(e)
	_, _ = w.Write(append(b, '\n'))
	return exitFor(e.Code)
}

func exitFor(c Code) int {
	switch c {
	case CodeBadInput:
		return ExitBadInput
	case CodeAuth:
		return ExitAuth
	case CodeNotFound:
		return ExitNotFound
	case CodeNeedsConfirm:
		return ExitNeedsConfirm
	case CodeConflict:
		return ExitConflict
	default:
		return ExitServer
	}
}
