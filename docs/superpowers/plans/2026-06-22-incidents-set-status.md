# incidents set-status Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `armoctl incidents set-status` to change incidents to any of the four backend statuses (primary: Dismissed), with bulk selection by GUID list, stdin, `--filter`, and `--search`; refactor `resolve` into a thin alias.

**Architecture:** A single shared helper `runStatusChange` builds the `POST /runtime/incidents/changeStatus` request, validates/normalizes status, and runs the existing `safety.Wrap` mutation contract. `resolve` and the new `set-status` command both route through it. No backend changes — the endpoint already supports all four statuses, multi-GUID arrays, `innerFilters`, and a `searchText` query param, all processed async.

**Tech Stack:** Go, cobra, existing `armoctl` packages (`cmd/cliclient`, `cmd/cliflags`, `internal/apiclient`, `internal/safety`, `internal/clierr`, `internal/skillmeta`).

## Global Constraints

- Module path: `github.com/armosec/armoctl`. Work from the `armoctl/` repo root.
- Branch: `feat/incidents-set-status` (already created).
- Canonical statuses (verbatim from `kdr.IncidentStatus`): `Open`, `Investigating`, `Dismissed`, `Resolved`.
- Endpoint path passed to `apiclient` is `/runtime/incidents/changeStatus` (client prepends `/api/v1`).
- `searchText` is a **URL query parameter**, not a body field.
- `apiclient.Client.Do` signature: `Do(ctx context.Context, method, path string, query url.Values, body any) (*http.Response, error)`.
- Preserve existing `resolve` behavior exactly; `cmd/incidents/resolve_test.go` must pass unchanged.
- Run tests from the `armoctl/` directory: `go test ./cmd/incidents/...`.
- No Co-Authored-By or AI-attribution lines in commits (user global rule).

---

## File Structure

- `cmd/incidents/changestatus.go` (NEW): `normalizeStatus`, `statusChangeOpts`, `runStatusChange`, `statusChangeArgsLog`.
- `cmd/incidents/changestatus_test.go` (NEW): unit tests for `normalizeStatus`.
- `cmd/incidents/resolve.go` (MODIFY): refactor `RunE` to call `runStatusChange`.
- `cmd/incidents/setstatus.go` (NEW): `SetStatusCmd`, `readGUIDs`, `parseFilters`.
- `cmd/incidents/setstatus_test.go` (NEW): command behavior tests.
- `cmd/incidents/incidents.go` (MODIFY): register `SetStatusCmd`.
- `cmd/incidents/skill.go` (MODIFY): status field notes + recipes.
- `cmd/incidents/types.go` (MODIFY): cheatsheet status text.
- `docs/features/incidents-set-status.md` (NEW): feature doc (docs-gate).

---

## Task 1: Shared status-change helper + resolve refactor

**Files:**
- Create: `cmd/incidents/changestatus.go`
- Create: `cmd/incidents/changestatus_test.go`
- Modify: `cmd/incidents/resolve.go`
- Test (regression): `cmd/incidents/resolve_test.go` (unchanged)

**Interfaces:**
- Produces:
  - `func normalizeStatus(s string) (string, error)` — lowercases/trims input, returns canonical status or `clierr.CodeBadInput`.
  - `type statusChangeOpts struct { status string; guids []string; filters []map[string]string; searchText string; falsePositive bool; commandName string }`
  - `func runStatusChange(cmd *cobra.Command, cli *apiclient.Client, o statusChangeOpts) error`
  - `const changeStatusPath = "/runtime/incidents/changeStatus"`

- [ ] **Step 1: Write the failing test for `normalizeStatus`**

Create `cmd/incidents/changestatus_test.go`:

```go
package incidents

import "testing"

func TestNormalizeStatus(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"Dismissed", "Dismissed", false},
		{"dismissed", "Dismissed", false},
		{"  resolved ", "Resolved", false},
		{"INVESTIGATING", "Investigating", false},
		{"open", "Open", false},
		{"", "", true},
		{"bogus", "", true},
	}
	for _, c := range cases {
		got, err := normalizeStatus(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("normalizeStatus(%q): expected error, got %q", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("normalizeStatus(%q): unexpected error %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("normalizeStatus(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/incidents/ -run TestNormalizeStatus`
Expected: FAIL — `undefined: normalizeStatus`.

- [ ] **Step 3: Create the helper file**

Create `cmd/incidents/changestatus.go`:

```go
package incidents

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/armosec/armoctl/internal/safety"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const changeStatusPath = "/runtime/incidents/changeStatus"

// validStatuses maps lowercased input to its canonical form.
var validStatuses = map[string]string{
	"open":          "Open",
	"investigating": "Investigating",
	"dismissed":     "Dismissed",
	"resolved":      "Resolved",
}

// normalizeStatus validates and canonicalizes a status string.
func normalizeStatus(s string) (string, error) {
	canon, ok := validStatuses[strings.ToLower(strings.TrimSpace(s))]
	if !ok {
		return "", &clierr.Error{
			Code: clierr.CodeBadInput,
			Msg:  fmt.Sprintf("invalid --status %q (valid: Open, Investigating, Dismissed, Resolved)", s),
		}
	}
	return canon, nil
}

type statusChangeOpts struct {
	status        string
	guids         []string
	filters       []map[string]string
	searchText    string
	falsePositive bool
	commandName   string
}

// runStatusChange builds and (via safety.Wrap) executes a changeStatus request.
func runStatusChange(cmd *cobra.Command, cli *apiclient.Client, o statusChangeOpts) error {
	if len(o.guids) == 0 && len(o.filters) == 0 && o.searchText == "" {
		return &clierr.Error{
			Code: clierr.CodeBadInput,
			Msg:  "no incidents selected: provide GUID(s), --filter, or --search",
		}
	}

	guids := o.guids
	if guids == nil {
		guids = []string{}
	}
	filters := o.filters
	if filters == nil {
		filters = []map[string]string{}
	}
	body := map[string]any{
		"status":                o.status,
		"incidentsGuids":        guids,
		"innerFilters":          filters,
		"markedAsFalsePositive": o.falsePositive,
	}

	var query url.Values
	preview := map[string]any{"method": "POST", "url": changeStatusPath, "body": body}
	if o.searchText != "" {
		query = url.Values{"searchText": {o.searchText}}
		preview["query"] = map[string]any{"searchText": o.searchText}
	}

	m := cliflags.ReadMutation(cmd)

	return safety.Wrap(cmd.Context(), safety.Args{
		Command: o.commandName,
		DryRun:  m.DryRun,
		Yes:     m.Yes,
		Tty:     term.IsTerminal(int(os.Stdin.Fd())),
		Stdout:  cmd.OutOrStdout(),
		Stderr:  cmd.ErrOrStderr(),
		Preview: preview,
		ArgsLog: statusChangeArgsLog(o),
		Exec: func(ctx context.Context) (any, safety.ExecMeta, error) {
			resp, err := cli.Do(ctx, http.MethodPost, changeStatusPath, query, body)
			if err != nil {
				return nil, safety.ExecMeta{}, err
			}
			defer func() { _ = resp.Body.Close() }()
			raw, _ := io.ReadAll(resp.Body)
			if resp.StatusCode >= 400 {
				return nil, safety.ExecMeta{}, &clierr.Error{
					Code:      clierr.CodeServer,
					Msg:       strings.TrimSpace(string(raw)),
					RequestID: resp.Header.Get("x-request-id"),
				}
			}
			var out map[string]any
			_ = json.Unmarshal(raw, &out)
			return out, safety.ExecMeta{
				URL:       "POST " + changeStatusPath,
				Status:    resp.StatusCode,
				RequestID: resp.Header.Get("x-request-id"),
			}, nil
		},
	})
}

// statusChangeArgsLog renders a redacted one-line audit summary of the selection.
func statusChangeArgsLog(o statusChangeOpts) string {
	parts := []string{"status=" + o.status}
	if len(o.guids) > 0 {
		parts = append(parts, fmt.Sprintf("guids=%d", len(o.guids)))
	}
	if len(o.filters) > 0 {
		parts = append(parts, fmt.Sprintf("filters=%v", o.filters))
	}
	if o.searchText != "" {
		parts = append(parts, "search="+o.searchText)
	}
	if o.falsePositive {
		parts = append(parts, "falsePositive=true")
	}
	return strings.Join(parts, " ")
}
```

- [ ] **Step 4: Run the helper unit test to verify it passes**

Run: `go test ./cmd/incidents/ -run TestNormalizeStatus`
Expected: PASS.

- [ ] **Step 5: Refactor `resolve.go` to use the helper**

Replace the body of `RunE` and drop the now-unused imports. The full new `cmd/incidents/resolve.go`:

```go
package incidents

import (
	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/spf13/cobra"
)

func ResolveCmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{
		Use:   "resolve [guid]",
		Short: "Resolve a runtime incident (sets status to Resolved)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "resolve requires a GUID"}
			}
			falsePositive, _ := cmd.Flags().GetBool("false-positive")
			return runStatusChange(cmd, clientFor(cmd), statusChangeOpts{
				status:        "Resolved",
				guids:         []string{args[0]},
				falsePositive: falsePositive,
				commandName:   "incidents.resolve",
			})
		},
	}
	c.Flags().Bool("false-positive", false, "Mark the incident as a false positive when resolving")
	return c
}
```

- [ ] **Step 6: Run the full incidents package tests (regression)**

Run: `go test ./cmd/incidents/...`
Expected: PASS — including the unchanged `TestResolve_DryRunDoesNotCallServer` and `TestResolve_YesPostsAndReportsChanged`.

- [ ] **Step 7: Commit**

```bash
git add cmd/incidents/changestatus.go cmd/incidents/changestatus_test.go cmd/incidents/resolve.go
git commit -m "refactor(incidents): extract shared changeStatus helper; resolve uses it"
```

---

## Task 2: set-status command

**Files:**
- Create: `cmd/incidents/setstatus.go`
- Create: `cmd/incidents/setstatus_test.go`
- Modify: `cmd/incidents/incidents.go`

**Interfaces:**
- Consumes (from Task 1): `normalizeStatus`, `statusChangeOpts`, `runStatusChange`.
- Produces:
  - `func SetStatusCmd(clientFor cliclient.ClientFor) *cobra.Command`
  - `func readGUIDs(r io.Reader) ([]string, error)`
  - `func parseFilters(pairs []string) ([]map[string]string, error)`

- [ ] **Step 1: Write failing tests for `parseFilters` and the command**

Create `cmd/incidents/setstatus_test.go`:

```go
package incidents

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/spf13/cobra"
)

func TestParseFilters(t *testing.T) {
	got, err := parseFilters([]string{"severity=Low", "clusterName=prod"})
	if err != nil {
		t.Fatal(err)
	}
	want := []map[string]string{{"severity": "Low", "clusterName": "prod"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseFilters = %v, want %v", got, want)
	}
	if _, err := parseFilters([]string{"bad-no-eq"}); err == nil {
		t.Fatal("expected error for filter without '='")
	}
	if got, _ := parseFilters(nil); got != nil {
		t.Fatalf("parseFilters(nil) = %v, want nil", got)
	}
}

// newRoot wires a fresh root command with mutation flags and set-status.
func newRoot(c *apiclient.Client) (*cobra.Command, *bytes.Buffer) {
	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	root.AddCommand(SetStatusCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	return root, &stdout
}

func TestSetStatus_DryRunBody(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(200)
		_, _ = w.Write([]byte("{}"))
	}))
	defer srv.Close()
	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})

	root, stdout := newRoot(c)
	root.SetArgs([]string{"set-status", "i1", "i2", "--status", "Dismissed", "--dry-run"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&hits) != 0 {
		t.Fatal("server called during dry-run")
	}
	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("output not JSON: %v: %q", err, stdout.String())
	}
	req, _ := got["request"].(map[string]any)
	body, _ := req["body"].(map[string]any)
	if body["status"] != "Dismissed" {
		t.Fatalf("status: %v", body["status"])
	}
	guids, _ := body["incidentsGuids"].([]any)
	if len(guids) != 2 || guids[0] != "i1" || guids[1] != "i2" {
		t.Fatalf("incidentsGuids: %v", body["incidentsGuids"])
	}
}

func TestSetStatus_FilterAndSearch(t *testing.T) {
	c := apiclient.New(apiclient.Config{BaseURL: "http://unused", AccessKey: "K", CustomerGUID: "G"})
	root, stdout := newRoot(c)
	root.SetArgs([]string{
		"set-status", "--status", "Dismissed",
		"--filter", "severity=Low", "--filter", "clusterName=prod",
		"--search", "nginx", "--dry-run",
	})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	_ = json.Unmarshal(stdout.Bytes(), &got)
	req, _ := got["request"].(map[string]any)
	body, _ := req["body"].(map[string]any)
	filters, _ := body["innerFilters"].([]any)
	if len(filters) != 1 {
		t.Fatalf("innerFilters: %v", body["innerFilters"])
	}
	f0, _ := filters[0].(map[string]any)
	if f0["severity"] != "Low" || f0["clusterName"] != "prod" {
		t.Fatalf("filter map: %v", f0)
	}
	q, _ := req["query"].(map[string]any)
	if q["searchText"] != "nginx" {
		t.Fatalf("query searchText: %v", req["query"])
	}
}

func TestSetStatus_Stdin(t *testing.T) {
	c := apiclient.New(apiclient.Config{BaseURL: "http://unused", AccessKey: "K", CustomerGUID: "G"})
	root, stdout := newRoot(c)
	root.SetIn(strings.NewReader("i1 i2\ni3\n"))
	root.SetArgs([]string{"set-status", "--status", "Resolved", "--stdin", "--dry-run"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	_ = json.Unmarshal(stdout.Bytes(), &got)
	req, _ := got["request"].(map[string]any)
	body, _ := req["body"].(map[string]any)
	guids, _ := body["incidentsGuids"].([]any)
	if len(guids) != 3 {
		t.Fatalf("expected 3 guids from stdin, got %v", body["incidentsGuids"])
	}
}

func TestSetStatus_InvalidStatus(t *testing.T) {
	c := apiclient.New(apiclient.Config{BaseURL: "http://unused", AccessKey: "K", CustomerGUID: "G"})
	root, _ := newRoot(c)
	root.SetArgs([]string{"set-status", "i1", "--status", "Bogus", "--dry-run"})
	if err := root.ExecuteContext(context.Background()); err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestSetStatus_NoSelection(t *testing.T) {
	c := apiclient.New(apiclient.Config{BaseURL: "http://unused", AccessKey: "K", CustomerGUID: "G"})
	root, _ := newRoot(c)
	root.SetArgs([]string{"set-status", "--status", "Dismissed", "--dry-run"})
	if err := root.ExecuteContext(context.Background()); err == nil {
		t.Fatal("expected error when no incidents selected")
	}
}

func TestSetStatus_YesPosts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/runtime/incidents/changeStatus" {
			t.Errorf("path: %s", r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["status"] != "Dismissed" {
			t.Errorf("body.status: %v", body["status"])
		}
		w.Header().Set("x-request-id", "req-1")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"changed":true}`))
	}))
	defer srv.Close()
	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})

	t.Setenv("ARMOCTL_AUDIT_LOG", t.TempDir()+"/audit.log")

	root, stdout := newRoot(c)
	root.SetArgs([]string{"set-status", "i1", "--status", "Dismissed", "--yes"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), `"changed": true`) {
		t.Fatalf("changed not set: %s", stdout.String())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/incidents/ -run TestSetStatus`
Expected: FAIL — `undefined: SetStatusCmd` / `parseFilters` / `readGUIDs`.

- [ ] **Step 3: Create the command**

Create `cmd/incidents/setstatus.go`:

```go
package incidents

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/spf13/cobra"
)

// SetStatusCmd builds `armoctl incidents set-status`.
func SetStatusCmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{
		Use:   "set-status [guid...]",
		Short: "Change the status of runtime incidents (Open|Investigating|Dismissed|Resolved)",
		Long: "Change the status of one or more runtime incidents. Select incidents by GUID " +
			"(positional args and/or --stdin), by --filter key=value, and/or by --search text. " +
			"At least one selection method is required.",
		RunE: func(cmd *cobra.Command, args []string) error {
			statusFlag, _ := cmd.Flags().GetString("status")
			status, err := normalizeStatus(statusFlag)
			if err != nil {
				return err
			}

			guids := append([]string{}, args...)
			if useStdin, _ := cmd.Flags().GetBool("stdin"); useStdin {
				stdinGUIDs, err := readGUIDs(cmd.InOrStdin())
				if err != nil {
					return err
				}
				guids = append(guids, stdinGUIDs...)
			}

			filterPairs, _ := cmd.Flags().GetStringArray("filter")
			filters, err := parseFilters(filterPairs)
			if err != nil {
				return err
			}

			search, _ := cmd.Flags().GetString("search")
			falsePositive, _ := cmd.Flags().GetBool("false-positive")

			return runStatusChange(cmd, clientFor(cmd), statusChangeOpts{
				status:        status,
				guids:         guids,
				filters:       filters,
				searchText:    strings.TrimSpace(search),
				falsePositive: falsePositive,
				commandName:   "incidents.set-status",
			})
		},
	}
	c.Flags().String("status", "", "Target status: Open|Investigating|Dismissed|Resolved (required)")
	c.Flags().StringArray("filter", nil, "Filter as key=value (repeatable); selects incidents to change")
	c.Flags().String("search", "", "Free-text search to select incidents")
	c.Flags().Bool("stdin", false, "Read additional incident GUIDs from stdin")
	c.Flags().Bool("false-positive", false, "Mark the incidents as false positives")
	return c
}

// readGUIDs reads whitespace/newline-separated GUIDs from r.
func readGUIDs(r io.Reader) ([]string, error) {
	var out []string
	sc := bufio.NewScanner(r)
	sc.Split(bufio.ScanWords)
	for sc.Scan() {
		if tok := strings.TrimSpace(sc.Text()); tok != "" {
			out = append(out, tok)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, &clierr.Error{Code: clierr.CodeBadInput, Msg: "failed to read GUIDs from stdin: " + err.Error()}
	}
	return out, nil
}

// parseFilters converts repeated key=value pairs into a single innerFilters map.
func parseFilters(pairs []string) ([]map[string]string, error) {
	if len(pairs) == 0 {
		return nil, nil
	}
	m := map[string]string{}
	for _, p := range pairs {
		k, v, ok := strings.Cut(p, "=")
		k = strings.TrimSpace(k)
		if !ok || k == "" {
			return nil, &clierr.Error{
				Code: clierr.CodeBadInput,
				Msg:  fmt.Sprintf("invalid --filter %q (expected key=value)", p),
			}
		}
		m[k] = strings.TrimSpace(v)
	}
	return []map[string]string{m}, nil
}
```

- [ ] **Step 4: Register the command**

Modify `cmd/incidents/incidents.go` — add the line after `ResolveCmd`:

```go
	c.AddCommand(ResolveCmd(clientFor))
	c.AddCommand(SetStatusCmd(clientFor))
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./cmd/incidents/...`
Expected: PASS — all `TestSetStatus*`, `TestParseFilters`, and the Task 1 / existing resolve tests.

- [ ] **Step 6: Commit**

```bash
git add cmd/incidents/setstatus.go cmd/incidents/setstatus_test.go cmd/incidents/incidents.go
git commit -m "feat(incidents): add set-status command with bulk selection"
```

---

## Task 3: Skill metadata + docs

**Files:**
- Modify: `cmd/incidents/skill.go`
- Modify: `cmd/incidents/types.go`
- Create: `docs/features/incidents-set-status.md`

**Interfaces:**
- Consumes: nothing new (metadata/text only).
- Produces: updated skill recipes and feature documentation.

- [ ] **Step 1: Update the status field note in `skill.go`**

In `cmd/incidents/skill.go`, replace the `attributes.incidentStatus` `FieldNotes` entry:

```go
			"attributes.incidentStatus": "Live state machine. Canonical values: " +
				"Open | Investigating | Dismissed | Resolved. " +
				"Access with path syntax: .attributes.incidentStatus",
```

- [ ] **Step 2: Add recipes for bulk dismiss in `skill.go`**

In `cmd/incidents/skill.go`, append to the `Recipes` slice:

```go
				{
					Title: "Dismiss incidents matching a filter",
					Body:  "```\narmoctl incidents set-status --status Dismissed --filter severity=Low\n```",
				},
				{
					Title: "Bulk dismiss a list of incident GUIDs from stdin",
					Body:  "```\narmoctl incidents list --severity Low -o json | jq -r '.items[].guid' | \\\n  armoctl incidents set-status --status Dismissed --stdin --yes\n```",
				},
```

- [ ] **Step 3: Update the cheatsheet status text in `types.go`**

In `cmd/incidents/types.go`, replace the `attributes.incidentStatus` cheatsheet entry:

```go
		{"attributes.incidentStatus", "Current status: Open | Investigating | Dismissed | Resolved. Access with path syntax."},
```

- [ ] **Step 4: Create the feature doc**

Create `docs/features/incidents-set-status.md`:

```markdown
# incidents set-status

`armoctl incidents set-status` changes the status of one or more runtime incidents.
It calls `POST /runtime/incidents/changeStatus`, which the backend processes
asynchronously (status updates are published to Pulsar; `searchText` is resolved to
incident GUIDs and cluster names are enriched server-side).

## Usage

```text
armoctl incidents set-status [guid...] --status <Open|Investigating|Dismissed|Resolved> [flags]
```

| Flag | Description |
|------|-------------|
| `--status` | Required. Target status (case-insensitive; normalized to canonical form). |
| `--filter key=value` | Repeatable. Builds one `innerFilters` map (AND across filters). |
| `--search` | Free-text; sent as the `searchText` query parameter. |
| `--stdin` | Read additional GUIDs from stdin (whitespace/newline separated). |
| `--false-positive` | Sets `markedAsFalsePositive`. |

At least one of GUIDs, `--filter`, or `--search` is required.

## Selection examples

```bash
# Specific incidents
armoctl incidents set-status i1 i2 --status Dismissed

# Everything matching a filter (one async call)
armoctl incidents set-status --status Dismissed --filter severity=Low --filter clusterName=prod

# Pipe a GUID list from list output
armoctl incidents list --severity Low -o json | jq -r '.items[].guid' | \
  armoctl incidents set-status --status Dismissed --stdin --yes
```

## Safety

`set-status` is a mutation: use `--dry-run` to preview the request without sending it,
and `--yes` to confirm in non-interactive mode. Actions are written to the audit log.

## Relationship to `resolve`

`armoctl incidents resolve <guid>` is a thin alias that sets status to `Resolved`
(with optional `--false-positive`). Both commands share one internal request path.
```

- [ ] **Step 5: Build and run all incidents tests**

Run: `go build ./... && go test ./cmd/incidents/...`
Expected: build succeeds; all tests PASS.

- [ ] **Step 6: Verify the command end-to-end via help**

Run: `go run . incidents set-status --help`
Expected: help text lists `--status`, `--filter`, `--search`, `--stdin`, `--false-positive`.

- [ ] **Step 7: Commit**

```bash
git add cmd/incidents/skill.go cmd/incidents/types.go docs/features/incidents-set-status.md
git commit -m "docs(incidents): document set-status; update skill recipes and cheatsheet"
```

---

## Self-Review Notes

- **Spec coverage:** command shape (Task 2) ✓; resolve alias (Task 1) ✓; all-statuses + validation (Task 1 `normalizeStatus`) ✓; GUID+stdin selection (Task 2) ✓; filter selection (Task 2 `parseFilters`) ✓; search→query param (Task 1 `runStatusChange`, Task 2) ✓; reuse safety only / no pre-flight count (Task 1) ✓; error handling — no-selection, invalid status, server>=400 (Task 1) ✓; testing list (Tasks 1–2) ✓; skill + docs (Task 3) ✓.
- **Type consistency:** `statusChangeOpts`, `runStatusChange`, `normalizeStatus`, `readGUIDs`, `parseFilters`, `changeStatusPath` used identically across tasks.
- **No placeholders:** all steps contain runnable code and exact commands.
```
