# Risk-Acceptance (`risks exceptions`) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `risks exceptions` subcommand group to `armoctl` so agents can list, fetch, create, update, and delete security-risk exception policies — the platform's "risk-acceptance" workflow that was scoped out of phase 1.

**Architecture:** New subcommands live under the existing `risks` cluster (no new top-level cluster). They mirror the vulns/posture exception patterns: read commands use `apiclient` directly; mutating commands wrap the request in `safety.Wrap` with `--dry-run`/`--yes`/TTY detection and reuse the `codeForStatus` + `extractAPIMessage` helpers (duplicated locally, like vulns/posture). Body shape is `armotypes.BaseExceptionPolicy` with `policyType="securityRiskExceptionPolicy"`. The handler accepts exactly one entry in `policyIDs`, so the CLI surfaces a single `--risk-id` flag.

**Tech Stack:** Go 1.21+, cobra/viper, existing `internal/apiclient` + `internal/safety` + `internal/output` packages, `httptest` for unit tests.

**Resolved API surface (read from `cadashboardbe/httphandlerv2/securityriskshandler.go`, since swagger has been unreliable):**

| Verb | Path | Notes |
|---|---|---|
| POST | `/securityrisks/exceptions/list` | V2 paginated list. Body: `{pageNum, pageSize, innerFilters?}`. Response: `{response: [...], total: {value: N}}`. |
| GET | `/securityrisks/exceptions/<guid>` | Single policy. |
| POST | `/securityrisks/exceptions/new` | Create. Body: `BaseExceptionPolicy`. Server enforces exactly one entry in `policyIDs`. Returns the created policy. |
| PUT | `/securityrisks/exceptions` | Update. Body: `BaseExceptionPolicy` with `guid` populated. Server enforces exactly one entry in `policyIDs`. |
| DELETE | `/securityrisks/exceptions/<guid>` | Delete a single policy by exception GUID. Returns `["deleted"]`. |

**`BaseExceptionPolicy` JSON fields used by the CLI:** `guid`, `name`, `policyType`, `policyIDs`, `creationTime`, `reason`, `expirationDate` (RFC3339), `resources` (`[]identifiers.PortalDesignator` — `{designatorType, attributes}`), `advancedScopes` (left out of v1), `createdBy`, `updatedTime`.

**Out of scope for this plan (deferred):**
- Multi-policyID exceptions (server rejects them).
- `advancedScopes` UX (low value for agents until we have a query language for them).
- Promoting `codeForStatus`/`extractAPIMessage` from vulns/posture into a shared package — done as a separate refactor PR if/when a third caller appears (this is the third caller, but the helper bodies are 30 lines each; deduping is a follow-up to keep this PR focused).

---

## File Structure

**New files (all under `cmd/risks/`):**
- `exceptions_list.go` — `ExceptionsListCmd` (POST `/securityrisks/exceptions/list`, paged)
- `exceptions_get.go` — `ExceptionsGetCmd` (GET `/securityrisks/exceptions/<guid>`)
- `exceptions_create.go` — `ExceptionsCreateCmd` (POST `/securityrisks/exceptions/new`, mutation)
- `exceptions_update.go` — `ExceptionsUpdateCmd` (PUT `/securityrisks/exceptions`, mutation)
- `exceptions_delete.go` — `ExceptionsDeleteCmd` (DELETE `/securityrisks/exceptions/<guid>`, mutation)
- `errors.go` — local `codeForStatus` + `extractAPIMessage` helpers (mirroring `cmd/vulns/types.go` lines 12–47)
- `exceptions_test.go` — table-driven tests for all five commands plus status-code mapping

**Modified files:**
- `cmd/risks/risks.go` — wire the new subcommand tree under `risks exceptions`
- `cmd/risks/types.go` — add `ExceptionSummary` slice + `"exceptions"` cheatsheet entries
- `SKILL.md` — add a "Accept a security risk" recipe (mirrors the `vulns exceptions create` recipe shape)
- `shared-designs-and-docs/armoctl-agent-bridge/2026-05-01-followups.md` — mark item 1 (risk acceptance) as resolved by this PR; leave items 2–4 untouched

---

## Task 1: Local error helpers

**Files:**
- Create: `cmd/risks/errors.go`

- [ ] **Step 1: Create the helpers file**

```go
package risks

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/armosec/armoctl/internal/clierr"
)

// codeForStatus returns the appropriate clierr.Code for an HTTP status code.
func codeForStatus(s int) clierr.Code {
	switch {
	case s == 401, s == 403:
		return clierr.CodeAuth
	case s == 404:
		return clierr.CodeNotFound
	case s == 409:
		return clierr.CodeConflict
	case s >= 400 && s < 500:
		return clierr.CodeBadInput
	default:
		return clierr.CodeServer
	}
}

// extractAPIMessage mirrors apiclient.mapHTTPError so commands that use
// cli.Do directly surface the same human-readable error text as the rest of the CLI.
func extractAPIMessage(body []byte, status int) string {
	var msg struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}
	_ = json.Unmarshal(body, &msg)
	if m := msg.Message; m != "" {
		return m
	}
	if m := msg.Error; m != "" {
		return m
	}
	if m := strings.TrimSpace(string(body)); m != "" {
		return m
	}
	return http.StatusText(status)
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./cmd/risks/...`
Expected: PASS (no callers yet but file should compile cleanly).

- [ ] **Step 3: Commit**

```bash
git add cmd/risks/errors.go
git commit -m "risks: add codeForStatus + extractAPIMessage helpers for upcoming exceptions commands"
```

---

## Task 2: Cheatsheet + summary projection

**Files:**
- Modify: `cmd/risks/types.go`

- [ ] **Step 1: Add `ExceptionSummary` and the `"exceptions"` cheatsheet entry**

Edit `cmd/risks/types.go`. Below the existing `var ResourceSummary = ...` line, add:

```go
var ExceptionSummary = []string{
	"guid", "name", "policyIDs", "reason", "expirationDate", "creationTime", "createdBy",
}
```

In `Cheatsheet()`, add a third map entry next to `"risks"` and `"resources"`:

```go
"exceptions": {
	{"guid", "Exception policy GUID; required for get/update/delete."},
	{"name", "Human-readable policy name."},
	{"policyIDs", "Security risk IDs covered by the exception (exactly one supported)."},
	{"reason", "Reason recorded when the risk was accepted."},
	{"expirationDate", "RFC3339 expiration; null = no expiry."},
	{"creationTime", "RFC3339 first-created time."},
	{"createdBy", "User that created the policy."},
	{"resources", "Optional resource scope (PortalDesignators)."},
},
```

- [ ] **Step 2: Build to confirm syntax**

Run: `go build ./cmd/risks/...`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add cmd/risks/types.go
git commit -m "risks: add exceptions cheatsheet and summary projection"
```

---

## Task 3: `risks exceptions list` — paginated list

**Files:**
- Create: `cmd/risks/exceptions_list.go`
- Test: `cmd/risks/exceptions_test.go`

- [ ] **Step 1: Write the failing test**

Append to `cmd/risks/exceptions_test.go` (creating the file if it doesn't exist — see Task 8 for the file header that earlier tests will assume; for this first test you can include the header now and keep extending in later tasks).

```go
package risks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/spf13/cobra"
)

func newExcRoot() (*cobra.Command, *bytes.Buffer) {
	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	return root, &stdout
}

func TestExceptionsList_PostsListWithPagination(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: got %s, want POST", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/securityrisks/exceptions/list") {
			t.Errorf("path: got %s", r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["pageNum"] == nil || body["pageSize"] == nil {
			t.Errorf("body missing pageNum/pageSize: %v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": []map[string]any{
				{"guid": "e1", "name": "accept-risk-1", "policyIDs": []string{"R-1"}},
				{"guid": "e2", "name": "accept-risk-2", "policyIDs": []string{"R-2"}},
			},
			"total": map[string]any{"value": 2},
		})
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root, stdout := newExcRoot()
	exc := &cobra.Command{Use: "exceptions"}
	exc.AddCommand(ExceptionsListCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	root.AddCommand(exc)
	root.SetArgs([]string{"exceptions", "list"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	out := stdout.String()
	if !strings.Contains(out, `"items"`) || !strings.Contains(out, "e1") || !strings.Contains(out, "e2") {
		t.Fatalf("unexpected list output: %s", out)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/risks/ -run TestExceptionsList_PostsListWithPagination -v`
Expected: FAIL with "ExceptionsListCmd undefined" (or similar).

- [ ] **Step 3: Implement `ExceptionsListCmd`**

Create `cmd/risks/exceptions_list.go`:

```go
package risks

import (
	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/armosec/armoctl/internal/output"
	"github.com/spf13/cobra"
)

func ExceptionsListCmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "List security-risk exception policies",
		RunE: func(cmd *cobra.Command, args []string) error {
			cli := clientFor(cmd)
			pg := cliflags.ReadPage(cmd)
			body := map[string]any{}
			if riskID, _ := cmd.Flags().GetString("risk-id"); riskID != "" {
				body["innerFilters"] = []map[string]string{{"policyIDs": riskID}}
			}
			res, err := cli.ListPaged(cmd.Context(), "/securityrisks/exceptions/list", nil, apiclient.ListOpts{
				Method:   "POST",
				Body:     body,
				Limit:    pg.Limit,
				Page:     pg.Page,
				PageSize: pg.PageSize,
			})
			if err != nil {
				return err
			}
			list := output.List{Items: res.Items, Total: res.Total, Page: res.Page, PageSize: res.PageSize, NextCursor: res.NextCursor}
			return output.Render(cmd.OutOrStdout(), list, cliflags.OutputOptions(cmd, ExceptionSummary))
		},
	}
	c.Flags().String("risk-id", "", "Filter exceptions to those covering this security risk ID")
	return c
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/risks/ -run TestExceptionsList_PostsListWithPagination -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/risks/exceptions_list.go cmd/risks/exceptions_test.go
git commit -m "risks: add 'exceptions list' subcommand"
```

---

## Task 4: `risks exceptions get` — fetch one by GUID

**Files:**
- Create: `cmd/risks/exceptions_get.go`
- Modify: `cmd/risks/exceptions_test.go`

- [ ] **Step 1: Write the failing test**

Append to `cmd/risks/exceptions_test.go`:

```go
func TestExceptionsGet_ByGUID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method: got %s, want GET", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/securityrisks/exceptions/abc-123") {
			t.Errorf("path: got %s, want suffix /securityrisks/exceptions/abc-123", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"guid": "abc-123", "name": "my-exception", "policyIDs": []string{"R-9"},
		})
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root, stdout := newExcRoot()
	exc := &cobra.Command{Use: "exceptions"}
	exc.AddCommand(ExceptionsGetCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	root.AddCommand(exc)
	root.SetArgs([]string{"exceptions", "get", "abc-123"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "abc-123") {
		t.Fatalf("output missing guid: %s", stdout.String())
	}
}

func TestExceptionsGet_NoArgFails(t *testing.T) {
	c := apiclient.New(apiclient.Config{BaseURL: "http://localhost", AccessKey: "K", CustomerGUID: "G"})
	root, _ := newExcRoot()
	exc := &cobra.Command{Use: "exceptions"}
	exc.AddCommand(ExceptionsGetCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	root.AddCommand(exc)
	root.SetArgs([]string{"exceptions", "get"})
	err := root.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error when guid arg missing")
	}
	var ce *clierr.Error
	if !errors.As(err, &ce) || ce.Code != clierr.CodeBadInput {
		t.Fatalf("expected CodeBadInput, got %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/risks/ -run TestExceptionsGet -v`
Expected: FAIL with "ExceptionsGetCmd undefined".

- [ ] **Step 3: Implement `ExceptionsGetCmd`**

Create `cmd/risks/exceptions_get.go`:

```go
package risks

import (
	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/armosec/armoctl/internal/output"
	"github.com/spf13/cobra"
)

func ExceptionsGetCmd(clientFor cliclient.ClientFor) *cobra.Command {
	return &cobra.Command{
		Use:   "get <guid>",
		Short: "Get a security-risk exception policy by GUID",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "get requires the exception GUID"}
			}
			cli := clientFor(cmd)
			var raw map[string]any
			if err := cli.GetJSON(cmd.Context(), "/securityrisks/exceptions/"+args[0], nil, &raw); err != nil {
				return err
			}
			return output.Render(cmd.OutOrStdout(), output.Get{Object: raw}, cliflags.OutputOptions(cmd, ExceptionSummary))
		},
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/risks/ -run TestExceptionsGet -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/risks/exceptions_get.go cmd/risks/exceptions_test.go
git commit -m "risks: add 'exceptions get' subcommand"
```

---

## Task 5: `risks exceptions create` — accept a risk

**Files:**
- Create: `cmd/risks/exceptions_create.go`
- Modify: `cmd/risks/exceptions_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `cmd/risks/exceptions_test.go`:

```go
func TestExceptionsCreate_DryRun(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root, stdout := newExcRoot()
	exc := &cobra.Command{Use: "exceptions"}
	exc.AddCommand(ExceptionsCreateCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	root.AddCommand(exc)
	root.SetArgs([]string{
		"exceptions", "create",
		"--name", "accept-r1",
		"--risk-id", "R-1",
		"--reason", "compensating-control",
		"--dry-run",
	})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&hits) != 0 {
		t.Errorf("server contacted during dry-run (%d hits)", hits)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("output not JSON: %v — %s", err, stdout.String())
	}
	dryRun, _ := result["dryRun"].(bool)
	if !dryRun {
		t.Errorf("expected dryRun=true: %v", result)
	}
	req, _ := result["request"].(map[string]any)
	body, _ := req["body"].(map[string]any)
	if body["policyType"] != "securityRiskExceptionPolicy" {
		t.Errorf("policyType: got %v", body["policyType"])
	}
	pids, _ := body["policyIDs"].([]any)
	if len(pids) != 1 || pids[0] != "R-1" {
		t.Errorf("policyIDs: got %v", body["policyIDs"])
	}
	if req["url"] != "/securityrisks/exceptions/new" {
		t.Errorf("url: got %v", req["url"])
	}
}

func TestExceptionsCreate_Yes(t *testing.T) {
	var captured map[string]any
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		if r.Method != http.MethodPost {
			t.Errorf("method: got %s, want POST", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/securityrisks/exceptions/new") {
			t.Errorf("path: got %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&captured)
		_ = json.NewEncoder(w).Encode(map[string]any{"guid": "new-guid", "name": "accept-r1", "policyIDs": []string{"R-1"}})
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root, stdout := newExcRoot()
	exc := &cobra.Command{Use: "exceptions"}
	exc.AddCommand(ExceptionsCreateCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	root.AddCommand(exc)
	root.SetArgs([]string{
		"exceptions", "create",
		"--name", "accept-r1",
		"--risk-id", "R-1",
		"--yes",
	})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&hits) != 1 {
		t.Errorf("expected 1 hit, got %d", hits)
	}
	if captured["policyType"] != "securityRiskExceptionPolicy" {
		t.Errorf("policyType: got %v", captured["policyType"])
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("output not JSON: %v — %s", err, stdout.String())
	}
	if changed, _ := result["changed"].(bool); !changed {
		t.Errorf("expected changed=true: %v", result)
	}
}

func TestExceptionsCreate_NoRiskIDFails(t *testing.T) {
	c := apiclient.New(apiclient.Config{BaseURL: "http://localhost", AccessKey: "K", CustomerGUID: "G"})
	root, _ := newExcRoot()
	exc := &cobra.Command{Use: "exceptions"}
	exc.AddCommand(ExceptionsCreateCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	root.AddCommand(exc)
	root.SetArgs([]string{"exceptions", "create", "--name", "x", "--yes"})
	err := root.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error when --risk-id missing")
	}
	var ce *clierr.Error
	if !errors.As(err, &ce) || ce.Code != clierr.CodeBadInput {
		t.Fatalf("expected CodeBadInput, got %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/risks/ -run TestExceptionsCreate -v`
Expected: FAIL with "ExceptionsCreateCmd undefined".

- [ ] **Step 3: Implement `ExceptionsCreateCmd`**

Create `cmd/risks/exceptions_create.go`:

```go
package risks

import (
	"context"
	"os"

	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/armosec/armoctl/internal/safety"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func ExceptionsCreateCmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a security-risk exception policy (accept a risk)",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			riskID, _ := cmd.Flags().GetString("risk-id")
			reason, _ := cmd.Flags().GetString("reason")
			expires, _ := cmd.Flags().GetString("expires")
			cluster, _ := cmd.Flags().GetString("cluster")
			namespace, _ := cmd.Flags().GetString("namespace")
			kind, _ := cmd.Flags().GetString("kind")
			workload, _ := cmd.Flags().GetString("workload")

			if riskID == "" {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "create requires --risk-id"}
			}

			body := map[string]any{
				"name":       name,
				"policyType": "securityRiskExceptionPolicy",
				"policyIDs":  []string{riskID},
			}
			if reason != "" {
				body["reason"] = reason
			}
			if expires != "" {
				body["expirationDate"] = expires
			}

			attrs := map[string]string{}
			if cluster != "" {
				attrs["cluster"] = cluster
			}
			if namespace != "" {
				attrs["namespace"] = namespace
			}
			if kind != "" {
				attrs["kind"] = kind
			}
			if workload != "" {
				attrs["name"] = workload
			}
			if len(attrs) > 0 {
				body["resources"] = []any{map[string]any{
					"designatorType": "Attributes",
					"attributes":     attrs,
				}}
			}

			const path = "/securityrisks/exceptions/new"
			cli := clientFor(cmd)
			m := cliflags.ReadMutation(cmd)

			return safety.Wrap(cmd.Context(), safety.Args{
				Command: "risks.exceptions.create",
				DryRun:  m.DryRun,
				Yes:     m.Yes,
				Tty:     term.IsTerminal(int(os.Stdin.Fd())),
				Stdout:  cmd.OutOrStdout(),
				Stderr:  cmd.ErrOrStderr(),
				Preview: map[string]any{"method": "POST", "url": path, "body": body},
				ArgsLog: "riskID=" + riskID,
				Exec: func(ctx context.Context) (any, safety.ExecMeta, error) {
					var resp map[string]any
					if err := cli.PostJSON(ctx, path, nil, body, &resp); err != nil {
						return nil, safety.ExecMeta{}, err
					}
					return resp, safety.ExecMeta{URL: "POST " + path, Status: 200}, nil
				},
			})
		},
	}
	c.Flags().String("name", "", "Policy name")
	c.Flags().String("risk-id", "", "Security risk ID to accept (required)")
	c.Flags().String("reason", "", "Reason for accepting the risk")
	c.Flags().String("expires", "", "Expiration date (RFC3339)")
	c.Flags().String("cluster", "", "Optional resource scope: cluster name")
	c.Flags().String("namespace", "", "Optional resource scope: namespace")
	c.Flags().String("kind", "", "Optional resource scope: workload kind")
	c.Flags().String("workload", "", "Optional resource scope: workload name")
	return c
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/risks/ -run TestExceptionsCreate -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/risks/exceptions_create.go cmd/risks/exceptions_test.go
git commit -m "risks: add 'exceptions create' subcommand"
```

---

## Task 6: `risks exceptions update`

**Files:**
- Create: `cmd/risks/exceptions_update.go`
- Modify: `cmd/risks/exceptions_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `cmd/risks/exceptions_test.go`:

```go
func TestExceptionsUpdate_DryRun(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root, stdout := newExcRoot()
	exc := &cobra.Command{Use: "exceptions"}
	exc.AddCommand(ExceptionsUpdateCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	root.AddCommand(exc)
	root.SetArgs([]string{
		"exceptions", "update",
		"--guid", "abc-123",
		"--risk-id", "R-1",
		"--reason", "extending-acceptance",
		"--dry-run",
	})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&hits) != 0 {
		t.Errorf("server contacted during dry-run (%d hits)", hits)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("output not JSON: %v — %s", err, stdout.String())
	}
	req, _ := result["request"].(map[string]any)
	body, _ := req["body"].(map[string]any)
	if body["guid"] != "abc-123" {
		t.Errorf("body.guid: got %v", body["guid"])
	}
	if req["method"] != "PUT" {
		t.Errorf("method: got %v", req["method"])
	}
}

func TestExceptionsUpdate_NoNameOmitsField(t *testing.T) {
	var captured map[string]any
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		_ = json.NewDecoder(r.Body).Decode(&captured)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root, _ := newExcRoot()
	exc := &cobra.Command{Use: "exceptions"}
	exc.AddCommand(ExceptionsUpdateCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	root.AddCommand(exc)
	root.SetArgs([]string{
		"exceptions", "update",
		"--guid", "abc-123",
		"--risk-id", "R-1",
		"--yes",
	})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&hits) != 1 {
		t.Errorf("expected 1 hit, got %d", hits)
	}
	if _, ok := captured["name"]; ok {
		t.Errorf("expected 'name' absent when --name not passed: %v", captured)
	}
	if captured["guid"] != "abc-123" {
		t.Errorf("guid: got %v", captured["guid"])
	}
}

func TestExceptionsUpdate_StatusCodeMapping(t *testing.T) {
	cases := []struct {
		status int
		want   clierr.Code
	}{
		{401, clierr.CodeAuth},
		{404, clierr.CodeNotFound},
		{409, clierr.CodeConflict},
		{400, clierr.CodeBadInput},
		{500, clierr.CodeServer},
	}
	for _, tc := range cases {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(`{"message":"upstream said no"}`))
			}))
			defer srv.Close()
			c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})

			root, _ := newExcRoot()
			exc := &cobra.Command{Use: "exceptions"}
			exc.AddCommand(ExceptionsUpdateCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
			root.AddCommand(exc)
			root.SetArgs([]string{
				"exceptions", "update",
				"--guid", "abc",
				"--risk-id", "R-1",
				"--yes",
			})
			err := root.ExecuteContext(context.Background())
			if err == nil {
				t.Fatal("expected error")
			}
			var ce *clierr.Error
			if !errors.As(err, &ce) {
				t.Fatalf("error not *clierr.Error: %v", err)
			}
			if ce.Code != tc.want {
				t.Fatalf("code: got %v, want %v", ce.Code, tc.want)
			}
			// 5xx triggers apiclient retry which closes the body; skip
			// message assertion in that case.
			if tc.status < 500 && ce.Msg != "upstream said no" {
				t.Errorf("msg: got %q, want extracted JSON message", ce.Msg)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/risks/ -run TestExceptionsUpdate -v`
Expected: FAIL with "ExceptionsUpdateCmd undefined".

- [ ] **Step 3: Implement `ExceptionsUpdateCmd`**

Create `cmd/risks/exceptions_update.go`:

```go
package risks

import (
	"context"
	"io"
	"net/http"
	"os"

	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/armosec/armoctl/internal/safety"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func ExceptionsUpdateCmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{
		Use:   "update",
		Short: "Update an existing security-risk exception policy",
		RunE: func(cmd *cobra.Command, args []string) error {
			guid, _ := cmd.Flags().GetString("guid")
			riskID, _ := cmd.Flags().GetString("risk-id")
			if guid == "" {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "update requires --guid"}
			}
			if riskID == "" {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "update requires --risk-id"}
			}

			// Optional fields included only when explicitly set, so users can
			// clear a field via --reason "".
			nameSet := cmd.Flags().Changed("name")
			reasonSet := cmd.Flags().Changed("reason")
			expiresSet := cmd.Flags().Changed("expires")
			name, _ := cmd.Flags().GetString("name")
			reason, _ := cmd.Flags().GetString("reason")
			expires, _ := cmd.Flags().GetString("expires")

			body := map[string]any{
				"guid":       guid,
				"policyType": "securityRiskExceptionPolicy",
				"policyIDs":  []string{riskID},
			}
			if nameSet {
				body["name"] = name
			}
			if reasonSet {
				body["reason"] = reason
			}
			if expiresSet {
				body["expirationDate"] = expires
			}

			const path = "/securityrisks/exceptions"
			cli := clientFor(cmd)
			m := cliflags.ReadMutation(cmd)

			return safety.Wrap(cmd.Context(), safety.Args{
				Command: "risks.exceptions.update",
				DryRun:  m.DryRun,
				Yes:     m.Yes,
				Tty:     term.IsTerminal(int(os.Stdin.Fd())),
				Stdout:  cmd.OutOrStdout(),
				Stderr:  cmd.ErrOrStderr(),
				Preview: map[string]any{"method": "PUT", "url": path, "body": body},
				ArgsLog: "guid=" + guid,
				Exec: func(ctx context.Context) (any, safety.ExecMeta, error) {
					resp, err := cli.Do(ctx, http.MethodPut, path, nil, body)
					if err != nil {
						return nil, safety.ExecMeta{}, err
					}
					defer func() { _ = resp.Body.Close() }()
					if resp.StatusCode >= 400 {
						b, _ := io.ReadAll(resp.Body)
						return nil, safety.ExecMeta{}, &clierr.Error{
							Code:      codeForStatus(resp.StatusCode),
							Msg:       extractAPIMessage(b, resp.StatusCode),
							RequestID: resp.Header.Get("x-request-id"),
						}
					}
					return map[string]any{"status": resp.StatusCode}, safety.ExecMeta{URL: "PUT " + path, Status: resp.StatusCode}, nil
				},
			})
		},
	}
	c.Flags().String("guid", "", "Exception policy GUID (required)")
	c.Flags().String("risk-id", "", "Security risk ID (required)")
	c.Flags().String("name", "", "Policy name")
	c.Flags().String("reason", "", "Reason")
	c.Flags().String("expires", "", "Expiration date (RFC3339)")
	return c
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/risks/ -run TestExceptionsUpdate -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/risks/exceptions_update.go cmd/risks/exceptions_test.go
git commit -m "risks: add 'exceptions update' subcommand"
```

---

## Task 7: `risks exceptions delete`

**Files:**
- Create: `cmd/risks/exceptions_delete.go`
- Modify: `cmd/risks/exceptions_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `cmd/risks/exceptions_test.go`:

```go
func TestExceptionsDelete_Yes(t *testing.T) {
	var capturedMethod, capturedPath string
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`["deleted"]`))
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root, stdout := newExcRoot()
	exc := &cobra.Command{Use: "exceptions"}
	exc.AddCommand(ExceptionsDeleteCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	root.AddCommand(exc)
	root.SetArgs([]string{"exceptions", "delete", "exc-guid", "--yes"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&hits) != 1 {
		t.Errorf("expected 1 hit, got %d", hits)
	}
	if capturedMethod != http.MethodDelete {
		t.Errorf("method: got %s, want DELETE", capturedMethod)
	}
	if !strings.HasSuffix(capturedPath, "/securityrisks/exceptions/exc-guid") {
		t.Errorf("path: got %s, want suffix /securityrisks/exceptions/exc-guid", capturedPath)
	}
	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("output not JSON: %v — %s", err, stdout.String())
	}
	if changed, _ := result["changed"].(bool); !changed {
		t.Errorf("expected changed=true: %v", result)
	}
}

func TestExceptionsDelete_NoArgFails(t *testing.T) {
	c := apiclient.New(apiclient.Config{BaseURL: "http://localhost", AccessKey: "K", CustomerGUID: "G"})
	root, _ := newExcRoot()
	exc := &cobra.Command{Use: "exceptions"}
	exc.AddCommand(ExceptionsDeleteCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	root.AddCommand(exc)
	root.SetArgs([]string{"exceptions", "delete", "--yes"})
	err := root.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error when guid arg missing")
	}
	var ce *clierr.Error
	if !errors.As(err, &ce) || ce.Code != clierr.CodeBadInput {
		t.Fatalf("expected CodeBadInput, got %v", err)
	}
}

func TestExceptionsDelete_StatusCodeMapping(t *testing.T) {
	cases := []struct {
		status int
		want   clierr.Code
	}{
		{401, clierr.CodeAuth},
		{404, clierr.CodeNotFound},
		{409, clierr.CodeConflict},
		{400, clierr.CodeBadInput},
		{500, clierr.CodeServer},
	}
	for _, tc := range cases {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(`{"error":"nope"}`))
			}))
			defer srv.Close()
			c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})

			root, _ := newExcRoot()
			exc := &cobra.Command{Use: "exceptions"}
			exc.AddCommand(ExceptionsDeleteCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
			root.AddCommand(exc)
			root.SetArgs([]string{"exceptions", "delete", "exc-guid", "--yes"})
			err := root.ExecuteContext(context.Background())
			if err == nil {
				t.Fatal("expected error")
			}
			var ce *clierr.Error
			if !errors.As(err, &ce) {
				t.Fatalf("error not *clierr.Error: %v", err)
			}
			if ce.Code != tc.want {
				t.Fatalf("code: got %v, want %v", ce.Code, tc.want)
			}
			if tc.status < 500 && ce.Msg != "nope" {
				t.Errorf("msg: got %q, want extracted JSON error", ce.Msg)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/risks/ -run TestExceptionsDelete -v`
Expected: FAIL with "ExceptionsDeleteCmd undefined".

- [ ] **Step 3: Implement `ExceptionsDeleteCmd`**

Create `cmd/risks/exceptions_delete.go`:

```go
package risks

import (
	"context"
	"io"
	"net/http"
	"os"

	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/armosec/armoctl/internal/safety"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func ExceptionsDeleteCmd(clientFor cliclient.ClientFor) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <guid>",
		Short: "Delete a security-risk exception policy by GUID",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "delete requires the exception GUID"}
			}
			guid := args[0]
			cli := clientFor(cmd)
			m := cliflags.ReadMutation(cmd)
			path := "/securityrisks/exceptions/" + guid

			return safety.Wrap(cmd.Context(), safety.Args{
				Command: "risks.exceptions.delete",
				DryRun:  m.DryRun,
				Yes:     m.Yes,
				Tty:     term.IsTerminal(int(os.Stdin.Fd())),
				Stdout:  cmd.OutOrStdout(),
				Stderr:  cmd.ErrOrStderr(),
				Preview: map[string]any{"method": "DELETE", "url": path},
				ArgsLog: "guid=" + guid,
				Exec: func(ctx context.Context) (any, safety.ExecMeta, error) {
					resp, err := cli.Do(ctx, http.MethodDelete, path, nil, nil)
					if err != nil {
						return nil, safety.ExecMeta{}, err
					}
					defer func() { _ = resp.Body.Close() }()
					if resp.StatusCode >= 400 {
						b, _ := io.ReadAll(resp.Body)
						return nil, safety.ExecMeta{}, &clierr.Error{
							Code:      codeForStatus(resp.StatusCode),
							Msg:       extractAPIMessage(b, resp.StatusCode),
							RequestID: resp.Header.Get("x-request-id"),
						}
					}
					return map[string]any{"deleted": guid}, safety.ExecMeta{URL: "DELETE " + path, Status: resp.StatusCode}, nil
				},
			})
		},
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/risks/ -run TestExceptionsDelete -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/risks/exceptions_delete.go cmd/risks/exceptions_test.go
git commit -m "risks: add 'exceptions delete' subcommand"
```

---

## Task 8: Wire the `exceptions` subcommand tree under `risks`

**Files:**
- Modify: `cmd/risks/risks.go`

- [ ] **Step 1: Add exceptions parent to the risks command tree**

Edit `cmd/risks/risks.go`. Replace the body of `Cmd` so it adds an `exceptions` subgroup:

```go
package risks

import (
	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/spf13/cobra"
)

func Cmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{Use: "risks", Short: "Inspect prioritized security risks"}
	c.AddCommand(FieldsCmd())
	c.AddCommand(ListCmd(clientFor))
	c.AddCommand(ResourcesCmd(clientFor))
	c.AddCommand(SeveritiesCmd(clientFor))

	exc := &cobra.Command{Use: "exceptions", Short: "Manage security-risk exception policies (risk acceptance)"}
	exc.AddCommand(ExceptionsListCmd(clientFor))
	exc.AddCommand(ExceptionsGetCmd(clientFor))
	exc.AddCommand(ExceptionsCreateCmd(clientFor))
	exc.AddCommand(ExceptionsUpdateCmd(clientFor))
	exc.AddCommand(ExceptionsDeleteCmd(clientFor))
	c.AddCommand(exc)

	return c
}
```

- [ ] **Step 2: Verify wiring with a smoke test**

Run: `go run . risks exceptions --help`
Expected: Help text lists `list`, `get`, `create`, `update`, `delete`.

Run: `go test ./cmd/risks/...`
Expected: All tests PASS.

- [ ] **Step 3: Commit**

```bash
git add cmd/risks/risks.go
git commit -m "risks: wire 'exceptions' subcommand group"
```

---

## Task 9: SKILL.md recipe + follow-up doc update

**Files:**
- Modify: `SKILL.md`
- Modify: `../shared-designs-and-docs/armoctl-agent-bridge/2026-05-01-followups.md`

- [ ] **Step 1: Add a SKILL.md recipe**

Add a new recipe block in SKILL.md alongside the existing `vulns exceptions create` example. Keep it terse:

```markdown
### Accept a security risk

```bash
# Find the risk you want to accept:
armoctl risks list --severity critical -o json | jq '.items[] | {id, name, severity}'

# Accept it (dry-run first to inspect the request body):
armoctl risks exceptions create --risk-id R-1234 --reason "compensating-control in place" --expires 2026-12-01T00:00:00Z --dry-run

# When the preview looks right, run with --yes:
armoctl risks exceptions create --risk-id R-1234 --reason "compensating-control in place" --expires 2026-12-01T00:00:00Z --yes

# List/inspect/delete:
armoctl risks exceptions list
armoctl risks exceptions get <guid>
armoctl risks exceptions delete <guid> --yes
```
```

- [ ] **Step 2: Mark the follow-up resolved**

In `shared-designs-and-docs/armoctl-agent-bridge/2026-05-01-followups.md`, prepend a status note to section 1:

```markdown
## 1. Risk acceptance APIs are not yet exposed

> **Status (2026-05-03):** Resolved by PR adding `risks exceptions list/get/create/update/delete`. The remaining text below is preserved for historical context.
```

- [ ] **Step 3: Commit**

```bash
git add SKILL.md ../shared-designs-and-docs/armoctl-agent-bridge/2026-05-01-followups.md
git commit -m "docs: add risk-acceptance recipe and mark followup #1 resolved"
```

---

## Task 10: End-to-end verification + PR prep

**Files:** none (verification only).

- [ ] **Step 1: Run the full test suite**

Run: `go test ./... -race`
Expected: PASS.

- [ ] **Step 2: Build the binary**

Run: `go build ./...`
Expected: PASS.

- [ ] **Step 3: Manual smoke against `api-dev.armosec.io`**

If credentials are available (`source ../armo-test-credentials.sh`):

```bash
armoctl risks exceptions list --limit 5
# pick a risk-id from `armoctl risks list -o json`
armoctl risks exceptions create --risk-id <id> --reason "smoke test" --dry-run
# only run with --yes if you intend to leave a real exception in the test tenant
```

Expected: list returns successfully (possibly empty); dry-run prints the preview without contacting the server twice.

- [ ] **Step 4: Open the PR**

```bash
git push -u origin feature/risks-exceptions-cluster
gh pr create --title "risks: risk-acceptance (exceptions) cluster" --body "$(cat <<'EOF'
## Summary
- Adds `risks exceptions list/get/create/update/delete` covering the security-risk acceptance flow gap noted in the phase-1 followups doc.
- Wraps create/update/delete with the standard mutation safety pattern (`--dry-run`/`--yes`/TTY check) used by vulns and posture exception clusters.
- Adds an "Accept a security risk" recipe to SKILL.md.

## Test plan
- [ ] `go test ./... -race` passes
- [ ] `armoctl risks exceptions --help` lists all five subcommands
- [ ] Live: `armoctl risks exceptions list --limit 5` against api-dev returns a paged result
- [ ] Live: `armoctl risks exceptions create --risk-id <id> --dry-run` prints a preview and does not contact the server
EOF
)"
```

Expected: PR opens cleanly against `main`; CI runs.

---

## Notes for the implementer

- **Pattern to mirror:** `cmd/vulns/exceptions_*.go` is the closest match. Each new file should look almost identical with the path/policyType swapped.
- **Don't `MarkFlagRequired`:** Validate required flags inside `RunE` and return `&clierr.Error{Code: clierr.CodeBadInput, ...}` so the exit code is correct (this came up multiple times in earlier Copilot reviews).
- **`policyIDs` is a single-element slice on the wire:** the server explicitly rejects multi-policy bodies (`if len(policyIDs) > 1: 400`). The CLI surfaces this as a single `--risk-id` flag.
- **Read endpoints don't need `safety.Wrap`:** only create/update/delete. Get/list use `apiclient` directly.
- **Don't promote the helpers yet:** `codeForStatus` + `extractAPIMessage` are duplicated in `cmd/vulns/types.go` and `cmd/posture/`. Adding a third copy is fine; deduping is a separate refactor PR best done after this one ships.
