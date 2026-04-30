package safety

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/armosec/armoctl/internal/clierr"
)

func TestWrap_DryRunDoesNotCallExec(t *testing.T) {
	var stdout, stderr bytes.Buffer
	called := false
	err := Wrap(context.Background(), Args{
		Command: "incidents.resolve",
		DryRun:  true,
		Yes:     false,
		Tty:     false,
		Stdout:  &stdout,
		Stderr:  &stderr,
		Preview: map[string]any{"method": "POST", "url": "/x", "body": map[string]any{"reason": "fp"}},
		Exec: func(ctx context.Context) (any, ExecMeta, error) {
			called = true
			return nil, ExecMeta{}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("Exec called during dry-run")
	}
	if !strings.Contains(stdout.String(), `"dryRun"`) {
		t.Fatalf("stdout missing dryRun: %s", stdout.String())
	}
}

func TestWrap_NonTTYWithoutYesFails(t *testing.T) {
	err := Wrap(context.Background(), Args{
		Command: "incidents.resolve",
		Tty:     false,
		Yes:     false,
		Exec:    func(ctx context.Context) (any, ExecMeta, error) { return nil, ExecMeta{}, nil },
	})
	var ce *clierr.Error
	if !errors.As(err, &ce) || ce.Code != clierr.CodeNeedsConfirm {
		t.Fatalf("err = %v, want NEEDS_CONFIRM", err)
	}
}

func TestWrap_YesRunsExec(t *testing.T) {
	var stdout bytes.Buffer
	called := false
	err := Wrap(context.Background(), Args{
		Command: "incidents.resolve",
		Yes:     true,
		Tty:     false,
		Stdout:  &stdout,
		Exec: func(ctx context.Context) (any, ExecMeta, error) {
			called = true
			return map[string]any{"ok": true}, ExecMeta{Status: 200, URL: "POST /x", RequestID: "r"}, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("Exec not called")
	}
	if !strings.Contains(stdout.String(), `"changed": true`) {
		t.Fatalf("stdout missing changed: %s", stdout.String())
	}
}
