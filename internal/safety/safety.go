package safety

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"

	"github.com/armosec/armoctl/internal/clierr"
)

// Args configures a Wrap call.
type Args struct {
	Command string         // dotted, e.g. "incidents.resolve"
	DryRun  bool
	Yes     bool
	Tty     bool
	Preview map[string]any // would-be request, printed when DryRun
	Exec    func(ctx context.Context) (any, ExecMeta, error)
	ArgsLog string         // already-redacted args for audit log
	Stdout  io.Writer
	Stderr  io.Writer
}

// ExecMeta captures the audit details from an executed mutation.
type ExecMeta struct {
	URL       string
	Status    int
	RequestID string
}

// Wrap implements the mutation safety contract.
func Wrap(ctx context.Context, a Args) error {
	if a.DryRun {
		if a.Stdout == nil {
			return &clierr.Error{Code: clierr.CodeBadInput, Msg: "safety.Wrap: missing Stdout"}
		}
		out := map[string]any{"dryRun": true, "request": a.Preview, "command": a.Command}
		return writeJSON(a.Stdout, out)
	}

	if !a.Yes && !a.Tty {
		return &clierr.Error{
			Code: clierr.CodeNeedsConfirm,
			Msg:  "mutation requires --yes in non-interactive mode",
			Hint: "re-run with --dry-run first, then --yes",
		}
	}

	if !a.Yes && a.Tty {
		ok, err := askConfirm(a.Stderr)
		if err != nil {
			return err
		}
		if !ok {
			return &clierr.Error{Code: clierr.CodeNeedsConfirm, Msg: "user declined"}
		}
	}

	if a.Stdout == nil {
		return &clierr.Error{Code: clierr.CodeBadInput, Msg: "safety.Wrap: missing Stdout"}
	}

	result, meta, err := a.Exec(ctx)
	if err != nil {
		return err
	}
	_ = AuditAppend(Entry{
		Command:   a.Command,
		URL:       meta.URL,
		Status:    meta.Status,
		RequestID: meta.RequestID,
		Args:      a.ArgsLog,
	})
	out := map[string]any{"result": result, "changed": true, "dryRun": false}
	return writeJSON(a.Stdout, out)
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// stdinReader is a swappable indirection for tests.
var stdinReader = func() io.Reader { return os.Stdin }

func askConfirm(w io.Writer) (bool, error) {
	if w != nil {
		_, _ = io.WriteString(w, "proceed? [y/N] ")
	}
	br := bufio.NewReader(stdinReader())
	line, err := br.ReadString('\n')
	if err != nil {
		return false, nil
	}
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes", nil
}
