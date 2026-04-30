# armoctl Foundation + Incidents Cluster — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the shared scaffolding (apiclient, output, safety, schema, errors, root flags, extended configure) and the `incidents` reference cluster end-to-end, so subsequent cluster plans can run in parallel against a stable contract.

**Architecture:** Layered: cobra CLI per cluster → `internal/apiclient` (auth + paging + retry + customerGUID injection) → `internal/output` (result envelope, renderers, `--fields`/`--query`/summary projection) → `internal/safety` (mutation wrap with dry-run/--yes/audit) → `internal/clierr` (exit codes + stderr-JSON) → `internal/schema` (embedded JSON schemas + `armoctl schema`). Cluster packages own only their commands; everything else is shared.

**Tech Stack:** Go 1.26, cobra/viper, charm/fang+huh, gojq, gopkg.in/yaml.v3, encoding/csv, olekukonko/tablewriter, net/http (+ httptest in tests).

**Spec:** `shared-designs-and-docs/armoctl-agent-bridge/2026-04-30-design.md` (commits `a8bd81d`, `f8128e4`).

---

## File Structure (created or modified)

Created:
- `internal/clierr/errors.go` — typed errors, exit codes, stderr-JSON writer
- `internal/clierr/errors_test.go`
- `internal/apiclient/client.go` — HTTP client, header + customerGUID injection
- `internal/apiclient/client_test.go`
- `internal/apiclient/paging.go` — auto-paging up to `--limit`
- `internal/apiclient/paging_test.go`
- `internal/apiclient/retry.go` — exp backoff on 429/5xx
- `internal/apiclient/retry_test.go`
- `internal/output/result.go` — `Result` types (List/Get/Mutation)
- `internal/output/render.go` — json/yaml/ndjson/table/csv renderers
- `internal/output/render_test.go`
- `internal/output/fields.go` — `--fields` projection + summary view + `--full`
- `internal/output/fields_test.go`
- `internal/output/query.go` — `--query` gojq post-processing
- `internal/output/query_test.go`
- `internal/safety/safety.go` — mutation wrap (dry-run/--yes/tty)
- `internal/safety/safety_test.go`
- `internal/safety/audit.go` — `~/.armoctl/audit.log` append
- `internal/safety/audit_test.go`
- `internal/schema/schema.go` — schema embed + `armoctl schema` cmd
- `internal/schema/data/.gitkeep`
- `internal/schema/schema_test.go`
- `cmd/cliflags/flags.go` — registers shared persistent flags & helpers
- `cmd/incidents/incidents.go` — root command for cluster
- `cmd/incidents/types.go` — incident response struct + summary fields + cheatsheet
- `cmd/incidents/list.go`, `list_test.go`
- `cmd/incidents/get.go`, `get_test.go`
- `cmd/incidents/alerts.go`, `alerts_test.go`
- `cmd/incidents/explain.go`, `explain_test.go`
- `cmd/incidents/resolve.go`, `resolve_test.go`
- `cmd/incidents/unresolve.go`
- `cmd/incidents/severities.go`
- `cmd/incidents/fields.go` — `armoctl incidents fields`
- `cmd/incidents/incidents_e2e_test.go` — end-to-end with httptest
- `scripts/gen-schemas.sh` — schema generator skeleton

Modified:
- `root.go` — register `incidents` and `schema` commands; register persistent flags via `cliflags`
- `internal/config/config.go` — change default `api-url` to `api.armosec.io`; add `Whoami()`

---

## Task 1: Set default API URL to `api.armosec.io` and add Whoami stub

**Files:**
- Modify: `internal/config/config.go`
- Modify: `root.go`
- Test: `internal/config/config_test.go` (create)

- [ ] **Step 1: Write the failing test**

`internal/config/config_test.go`:
```go
package config

import (
	"testing"

	"github.com/spf13/viper"
)

func TestDefaultAPIURL(t *testing.T) {
	viper.Reset()
	ApplyDefaults()
	if got := viper.GetString("api-url"); got != "api.armosec.io" {
		t.Fatalf("api-url default = %q, want %q", got, "api.armosec.io")
	}
}
```

- [ ] **Step 2: Run test — expect FAIL**

Run: `go test ./internal/config/... -run TestDefaultAPIURL -v`
Expected: FAIL — `ApplyDefaults` undefined.

- [ ] **Step 3: Implement minimal change**

Add to `internal/config/config.go`:
```go
// ApplyDefaults installs viper defaults the rest of armoctl assumes.
// Safe to call multiple times.
func ApplyDefaults() {
	viper.SetDefault("api-url", "api.armosec.io")
}
```

Also change in `SaveConfig`:
```go
if v := viper.GetString("api-url"); v != "" && v != "api.armosec.io" {
```
(was `cloud.armosec.io`).

In `root.go`, replace `viper.SetDefault("api-url", "cloud.armosec.io")` with `config.ApplyDefaults()`.

- [ ] **Step 4: Run test — expect PASS**

Run: `go test ./internal/config/... -run TestDefaultAPIURL -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go root.go
git commit -m "config: default api-url to api.armosec.io and add ApplyDefaults"
```

---

## Task 2: `internal/clierr` — typed errors, exit codes, stderr-JSON writer

**Files:**
- Create: `internal/clierr/errors.go`
- Test: `internal/clierr/errors_test.go`

- [ ] **Step 1: Write the failing test**

```go
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
```

- [ ] **Step 2: Run test — expect FAIL**

Run: `go test ./internal/clierr/... -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Implement**

`internal/clierr/errors.go`:
```go
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
```

- [ ] **Step 4: Run test — expect PASS**

Run: `go test ./internal/clierr/... -v`
Expected: PASS (both tests).

- [ ] **Step 5: Commit**

```bash
git add internal/clierr
git commit -m "clierr: typed errors, exit codes, stderr-JSON writer"
```

---

## Task 3: `internal/apiclient` — base client with auth + customerGUID injection

**Files:**
- Create: `internal/apiclient/client.go`
- Test: `internal/apiclient/client_test.go`

- [ ] **Step 1: Write the failing test**

```go
package apiclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDoInjectsAuthAndCustomerGUID(t *testing.T) {
	var gotKey, gotGUID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("x-api-key")
		gotGUID = r.URL.Query().Get("customerGUID")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	resp, err := c.Do(context.Background(), "GET", "/runtime/incidents", nil, nil)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	resp.Body.Close()
	if gotKey != "K" {
		t.Fatalf("x-api-key = %q, want K", gotKey)
	}
	if gotGUID != "G" {
		t.Fatalf("customerGUID = %q, want G", gotGUID)
	}
}
```

- [ ] **Step 2: Run test — expect FAIL**

Run: `go test ./internal/apiclient/... -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Implement**

`internal/apiclient/client.go`:
```go
// Package apiclient is the shared HTTP client for the ARMO platform API.
package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/armosec/armoctl/internal/clierr"
)

type Config struct {
	BaseURL      string // e.g. "https://api.armosec.io" or "https://api.armosec.io/api/v1"
	AccessKey    string
	CustomerGUID string
	HTTPClient   *http.Client
}

type Client struct {
	cfg Config
	hc  *http.Client
}

func New(cfg Config) *Client {
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{cfg: cfg, hc: hc}
}

// Do issues a request to path (path may be absolute or relative to BaseURL).
// query params are merged onto the URL; customerGUID is injected automatically.
func (c *Client) Do(ctx context.Context, method, path string, query url.Values, body any) (*http.Response, error) {
	u, err := c.resolveURL(path, query)
	if err != nil {
		return nil, err
	}

	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		rdr = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), rdr)
	if err != nil {
		return nil, err
	}
	if c.cfg.AccessKey == "" || c.cfg.CustomerGUID == "" {
		return nil, &clierr.Error{Code: clierr.CodeAuth, Msg: "missing customer-guid or access-key", Hint: "run: armoctl configure"}
	}
	req.Header.Set("x-api-key", c.cfg.AccessKey)
	if body != nil {
		req.Header.Set("content-type", "application/json")
	}
	req.Header.Set("accept", "application/json")
	return c.hc.Do(req)
}

func (c *Client) resolveURL(path string, query url.Values) (*url.URL, error) {
	base := c.cfg.BaseURL
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		base = "https://" + base
	}
	if !strings.Contains(strings.TrimPrefix(strings.TrimPrefix(base, "https://"), "http://"), "/api/v") {
		base = strings.TrimRight(base, "/") + "/api/v1"
	}
	u, err := url.Parse(base + path)
	if err != nil {
		return nil, fmt.Errorf("resolving %s: %w", path, err)
	}
	q := u.Query()
	for k, vs := range query {
		for _, v := range vs {
			q.Add(k, v)
		}
	}
	q.Set("customerGUID", c.cfg.CustomerGUID)
	u.RawQuery = q.Encode()
	return u, nil
}
```

- [ ] **Step 4: Run test — expect PASS**

Run: `go test ./internal/apiclient/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/apiclient/client.go internal/apiclient/client_test.go
git commit -m "apiclient: base client with x-api-key and customerGUID injection"
```

---

## Task 4: `apiclient` — typed JSON helpers + status→error mapping

**Files:**
- Modify: `internal/apiclient/client.go`
- Test: `internal/apiclient/client_test.go`

- [ ] **Step 1: Write the failing test**

Append to `client_test.go`:
```go
func TestGetJSON_404IsNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-request-id", "req-1")
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"message":"nope"}`))
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	var out struct{ X int }
	err := c.GetJSON(context.Background(), "/runtime/incidents/abc", nil, &out)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	var ce *clierr.Error
	if !errors.As(err, &ce) || ce.Code != clierr.CodeNotFound {
		t.Fatalf("err = %v, want CodeNotFound", err)
	}
	if ce.RequestID != "req-1" {
		t.Fatalf("RequestID = %q, want req-1", ce.RequestID)
	}
}

func TestGetJSON_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"x":42}`))
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	var out struct{ X int }
	if err := c.GetJSON(context.Background(), "/x", nil, &out); err != nil {
		t.Fatal(err)
	}
	if out.X != 42 {
		t.Fatalf("out.X = %d", out.X)
	}
}
```

Add `"errors"` to test imports.

- [ ] **Step 2: Run test — expect FAIL**

Run: `go test ./internal/apiclient/... -v`
Expected: FAIL — `GetJSON` undefined.

- [ ] **Step 3: Implement**

Append to `internal/apiclient/client.go`:
```go
// GetJSON does a GET and decodes JSON into out.
func (c *Client) GetJSON(ctx context.Context, path string, query url.Values, out any) error {
	resp, err := c.Do(ctx, http.MethodGet, path, query, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decode(resp, out)
}

// PostJSON does a POST with JSON body and decodes JSON into out (out may be nil).
func (c *Client) PostJSON(ctx context.Context, path string, query url.Values, body, out any) error {
	resp, err := c.Do(ctx, http.MethodPost, path, query, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return decode(resp, out)
}

func decode(resp *http.Response, out any) error {
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return mapHTTPError(resp.StatusCode, resp.Header.Get("x-request-id"), body)
	}
	if out == nil || len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, out)
}

func mapHTTPError(status int, reqID string, body []byte) error {
	var msg struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}
	_ = json.Unmarshal(body, &msg)
	m := msg.Message
	if m == "" {
		m = msg.Error
	}
	if m == "" {
		m = strings.TrimSpace(string(body))
	}
	if m == "" {
		m = http.StatusText(status)
	}
	code := clierr.CodeServer
	switch {
	case status == 401, status == 403:
		code = clierr.CodeAuth
	case status == 404:
		code = clierr.CodeNotFound
	case status == 409:
		code = clierr.CodeConflict
	case status >= 400 && status < 500:
		code = clierr.CodeBadInput
	}
	return &clierr.Error{Code: code, Msg: m, RequestID: reqID}
}
```

- [ ] **Step 4: Run test — expect PASS**

Run: `go test ./internal/apiclient/... -v`
Expected: PASS (all 3 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/apiclient/client.go internal/apiclient/client_test.go
git commit -m "apiclient: GetJSON/PostJSON helpers and HTTP→clierr mapping"
```

---

## Task 5: `apiclient` — retry on 429/5xx with exponential backoff

**Files:**
- Create: `internal/apiclient/retry.go`
- Test: `internal/apiclient/retry_test.go`

- [ ] **Step 1: Write the failing test**

```go
package apiclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestDoRetriesOn429(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&hits, 1)
		if n < 3 {
			w.WriteHeader(429)
			return
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	c.retry = retryConfig{Max: 3, Base: 1 * time.Millisecond}
	resp, err := c.Do(context.Background(), "GET", "/x", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if got := atomic.LoadInt32(&hits); got != 3 {
		t.Fatalf("hits = %d, want 3", got)
	}
}

func TestDoGivesUpAfterMax(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	c.retry = retryConfig{Max: 2, Base: 1 * time.Millisecond}
	resp, err := c.Do(context.Background(), "GET", "/x", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 503 {
		t.Fatalf("status = %d, want 503", resp.StatusCode)
	}
}
```

- [ ] **Step 2: Run test — expect FAIL**

Run: `go test ./internal/apiclient/... -v`
Expected: FAIL — `retry`/`retryConfig` undefined.

- [ ] **Step 3: Implement**

`internal/apiclient/retry.go`:
```go
package apiclient

import (
	"math/rand/v2"
	"time"
)

type retryConfig struct {
	Max  int           // total attempts incl. first
	Base time.Duration // first backoff
}

var defaultRetry = retryConfig{Max: 3, Base: 200 * time.Millisecond}

func (rc retryConfig) sleepFor(attempt int) time.Duration {
	d := rc.Base << attempt
	jitter := time.Duration(rand.Int64N(int64(rc.Base)))
	return d + jitter
}
```

Modify `client.go` `Do` to wrap the request loop and add a `retry` field on `Client`:
```go
type Client struct {
	cfg   Config
	hc    *http.Client
	retry retryConfig
}

func New(cfg Config) *Client {
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{cfg: cfg, hc: hc, retry: defaultRetry}
}
```

Replace the existing `c.hc.Do(req)` in `Do` with:
```go
return c.doWithRetry(req)
```

Add helper at the bottom of `client.go`:
```go
func (c *Client) doWithRetry(req *http.Request) (*http.Response, error) {
	var lastErr error
	for attempt := 0; attempt < c.retry.Max; attempt++ {
		if attempt > 0 {
			select {
			case <-req.Context().Done():
				return nil, req.Context().Err()
			case <-time.After(c.retry.sleepFor(attempt - 1)):
			}
		}
		resp, err := c.hc.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode == 429 || resp.StatusCode >= 500 {
			resp.Body.Close()
			lastErr = nil
			continue
		}
		return resp, nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	// Last attempt's response was a 429/5xx — re-issue once and return whatever we get.
	return c.hc.Do(req)
}
```

- [ ] **Step 4: Run test — expect PASS**

Run: `go test ./internal/apiclient/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/apiclient/retry.go internal/apiclient/retry_test.go internal/apiclient/client.go
git commit -m "apiclient: retry on 429/5xx with exponential backoff"
```

---

## Task 6: `internal/output` — Result types + JSON renderer

**Files:**
- Create: `internal/output/result.go`
- Create: `internal/output/render.go`
- Test: `internal/output/render_test.go`

- [ ] **Step 1: Write the failing test**

```go
package output

import (
	"bytes"
	"encoding/json"
	"testing"
)

type item struct {
	Guid string `json:"guid"`
	Name string `json:"name"`
}

func TestRenderJSONList(t *testing.T) {
	r := List{
		Items:    []any{item{"a", "alpha"}, item{"b", "beta"}},
		Total:    2,
		Page:     1,
		PageSize: 50,
	}
	var buf bytes.Buffer
	if err := Render(&buf, r, Options{Format: "json"}); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("not JSON: %v: %q", err, buf.String())
	}
	if got["total"].(float64) != 2 {
		t.Fatalf("total = %v", got["total"])
	}
	items := got["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("items len = %d", len(items))
	}
}

func TestRenderJSONGet(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, Get{Object: item{"x", "ex"}}, Options{Format: "json"}); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("not JSON: %v: %q", err, buf.String())
	}
	if got["guid"] != "x" {
		t.Fatalf("guid = %v", got["guid"])
	}
}

func TestRenderJSONMutation(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, Mutation{Result: "ok", Changed: true}, Options{Format: "json"}); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("not JSON: %v: %q", err, buf.String())
	}
	if got["changed"] != true {
		t.Fatalf("changed = %v", got["changed"])
	}
}
```

- [ ] **Step 2: Run test — expect FAIL**

Run: `go test ./internal/output/... -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Implement**

`internal/output/result.go`:
```go
// Package output renders armoctl results in the formats agents and humans need.
package output

// Result is the marker interface for the three result shapes.
type Result interface{ isResult() }

// List is the envelope every list command produces.
type List struct {
	Items      []any  `json:"items"`
	Total      int    `json:"total"`
	Page       int    `json:"page,omitempty"`
	PageSize   int    `json:"pageSize,omitempty"`
	NextCursor string `json:"nextCursor,omitempty"`
}

func (List) isResult() {}

// Get wraps a single resource object.
type Get struct {
	Object any `json:"-"`
}

func (Get) isResult() {}

// Mutation is the standard result of any mutating command.
type Mutation struct {
	Result  any  `json:"result,omitempty"`
	Changed bool `json:"changed"`
	DryRun  bool `json:"dryRun"`
}

func (Mutation) isResult() {}

// Options controls rendering.
type Options struct {
	Format string // json | yaml | ndjson | table | csv
	Query  string // gojq expression
	Fields []string
	Full   bool
}
```

`internal/output/render.go`:
```go
package output

import (
	"encoding/json"
	"fmt"
	"io"
)

// Render writes r in the requested format.
func Render(w io.Writer, r Result, o Options) error {
	switch o.Format {
	case "", "json":
		return renderJSON(w, r)
	default:
		return fmt.Errorf("unsupported output format %q", o.Format)
	}
}

func renderJSON(w io.Writer, r Result) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	switch v := r.(type) {
	case Get:
		return enc.Encode(v.Object)
	default:
		return enc.Encode(r)
	}
}
```

- [ ] **Step 4: Run test — expect PASS**

Run: `go test ./internal/output/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/output
git commit -m "output: result types and JSON renderer"
```

---

## Task 7: `output` — YAML, NDJSON, CSV, table renderers

**Files:**
- Modify: `internal/output/render.go`
- Test: `internal/output/render_test.go`

- [ ] **Step 1: Write the failing test**

Append to `render_test.go`:
```go
import (
	"strings"
)

func TestRenderYAMLList(t *testing.T) {
	r := List{Items: []any{item{"a", "alpha"}}, Total: 1}
	var buf bytes.Buffer
	if err := Render(&buf, r, Options{Format: "yaml"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "items:") || !strings.Contains(buf.String(), "alpha") {
		t.Fatalf("yaml unexpected:\n%s", buf.String())
	}
}

func TestRenderNDJSONList(t *testing.T) {
	r := List{Items: []any{item{"a", "alpha"}, item{"b", "beta"}}, Total: 2}
	var buf, errBuf bytes.Buffer
	if err := Render(&buf, r, Options{Format: "ndjson", Stderr: &errBuf}); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("ndjson lines = %d: %q", len(lines), buf.String())
	}
	if !strings.Contains(errBuf.String(), `"total":2`) {
		t.Fatalf("ndjson stderr meta missing total: %q", errBuf.String())
	}
}

func TestRenderCSVList(t *testing.T) {
	r := List{Items: []any{item{"a", "alpha"}, item{"b", "beta"}}, Total: 2}
	var buf bytes.Buffer
	if err := Render(&buf, r, Options{Format: "csv"}); err != nil {
		t.Fatal(err)
	}
	csv := buf.String()
	if !strings.HasPrefix(csv, "guid,name\n") {
		t.Fatalf("csv header bad: %q", csv)
	}
	if !strings.Contains(csv, "a,alpha\n") {
		t.Fatalf("csv row missing: %q", csv)
	}
}

func TestRenderTableList(t *testing.T) {
	r := List{Items: []any{item{"a", "alpha"}}, Total: 1}
	var buf bytes.Buffer
	if err := Render(&buf, r, Options{Format: "table"}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "alpha") {
		t.Fatalf("table missing alpha:\n%s", out)
	}
}
```

- [ ] **Step 2: Run test — expect FAIL**

Run: `go test ./internal/output/... -v`
Expected: FAIL — these formats not handled; `Options.Stderr` undefined.

- [ ] **Step 3: Implement**

Modify `internal/output/result.go`:
```go
import "io"

type Options struct {
	Format string
	Query  string
	Fields []string
	Full   bool
	Stderr io.Writer // used by ndjson for envelope metadata
}
```

Replace `internal/output/render.go`:
```go
package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

func Render(w io.Writer, r Result, o Options) error {
	switch o.Format {
	case "", "json":
		return renderJSON(w, r)
	case "yaml":
		return renderYAML(w, r)
	case "ndjson":
		return renderNDJSON(w, r, o)
	case "csv":
		return renderCSV(w, r)
	case "table":
		return renderTable(w, r)
	default:
		return fmt.Errorf("unsupported output format %q", o.Format)
	}
}

func renderJSON(w io.Writer, r Result) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if g, ok := r.(Get); ok {
		return enc.Encode(g.Object)
	}
	return enc.Encode(r)
}

func renderYAML(w io.Writer, r Result) error {
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	defer enc.Close()
	if g, ok := r.(Get); ok {
		return enc.Encode(g.Object)
	}
	return enc.Encode(r)
}

func renderNDJSON(w io.Writer, r Result, o Options) error {
	switch v := r.(type) {
	case List:
		enc := json.NewEncoder(w)
		for _, it := range v.Items {
			if err := enc.Encode(it); err != nil {
				return err
			}
		}
		if o.Stderr != nil {
			meta := map[string]any{"total": v.Total, "page": v.Page, "pageSize": v.PageSize, "nextCursor": v.NextCursor}
			b, _ := json.Marshal(meta)
			fmt.Fprintln(o.Stderr, string(b))
		}
		return nil
	default:
		return renderJSON(w, r)
	}
}

func renderCSV(w io.Writer, r Result) error {
	v, ok := r.(List)
	if !ok {
		return fmt.Errorf("csv only supports list results")
	}
	if len(v.Items) == 0 {
		return nil
	}
	cols := flatColumns(v.Items[0])
	cw := csv.NewWriter(w)
	defer cw.Flush()
	if err := cw.Write(cols); err != nil {
		return err
	}
	for _, it := range v.Items {
		row, err := flatRow(it, cols)
		if err != nil {
			return err
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	return nil
}

func renderTable(w io.Writer, r Result) error {
	v, ok := r.(List)
	if !ok {
		return renderJSON(w, r)
	}
	if len(v.Items) == 0 {
		fmt.Fprintln(w, "(empty)")
		return nil
	}
	cols := flatColumns(v.Items[0])
	widths := make([]int, len(cols))
	rows := make([][]string, 0, len(v.Items))
	for i, c := range cols {
		widths[i] = len(c)
	}
	for _, it := range v.Items {
		row, err := flatRow(it, cols)
		if err != nil {
			return err
		}
		for i, cell := range row {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
		rows = append(rows, row)
	}
	writeRow(w, cols, widths)
	sep := make([]string, len(cols))
	for i := range sep {
		sep[i] = strings.Repeat("-", widths[i])
	}
	writeRow(w, sep, widths)
	for _, row := range rows {
		writeRow(w, row, widths)
	}
	return nil
}

func writeRow(w io.Writer, cells []string, widths []int) {
	parts := make([]string, len(cells))
	for i, c := range cells {
		parts[i] = fmt.Sprintf("%-*s", widths[i], c)
	}
	fmt.Fprintln(w, strings.Join(parts, "  "))
}

func flatColumns(item any) []string {
	m := toFlatMap(item)
	cols := make([]string, 0, len(m))
	for k := range m {
		cols = append(cols, k)
	}
	sort.Strings(cols)
	return cols
}

func flatRow(item any, cols []string) ([]string, error) {
	m := toFlatMap(item)
	row := make([]string, len(cols))
	for i, c := range cols {
		row[i] = fmt.Sprintf("%v", m[c])
	}
	return row, nil
}

// toFlatMap converts an item (struct or map) into a flat map[string]any
// using JSON tags as keys. Non-flat fields are JSON-encoded.
func toFlatMap(item any) map[string]any {
	b, err := json.Marshal(item)
	if err != nil {
		return map[string]any{}
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return map[string]any{}
	}
	out := map[string]any{}
	for k, v := range m {
		switch v.(type) {
		case map[string]any, []any:
			b2, _ := json.Marshal(v)
			out[k] = string(b2)
		default:
			out[k] = v
		}
	}
	_ = reflect.TypeOf
	return out
}
```

Add `gopkg.in/yaml.v3` to go.mod: `go get gopkg.in/yaml.v3`.

- [ ] **Step 4: Run test — expect PASS**

Run: `go test ./internal/output/... -v`
Expected: PASS (all 5 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/output go.mod go.sum
git commit -m "output: yaml, ndjson, csv, table renderers"
```

---

## Task 8: `output` — `--query` (gojq) post-processing

**Files:**
- Create: `internal/output/query.go`
- Test: `internal/output/query_test.go`
- Modify: `internal/output/render.go`

- [ ] **Step 1: Write the failing test**

```go
package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderWithQuery_OnList(t *testing.T) {
	r := List{Items: []any{
		map[string]any{"guid": "a", "severity": "high"},
		map[string]any{"guid": "b", "severity": "low"},
	}, Total: 2}
	var buf bytes.Buffer
	o := Options{Format: "json", Query: `.items[] | select(.severity=="high") | .guid`}
	if err := Render(&buf, r, o); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"a"`) || strings.Contains(buf.String(), `"b"`) {
		t.Fatalf("query result unexpected: %q", buf.String())
	}
}
```

- [ ] **Step 2: Run test — expect FAIL**

Run: `go test ./internal/output/... -v`
Expected: FAIL — `Query` is not yet applied.

- [ ] **Step 3: Implement**

Add `github.com/itchyny/gojq` (run `go get github.com/itchyny/gojq`).

`internal/output/query.go`:
```go
package output

import (
	"encoding/json"
	"fmt"

	"github.com/itchyny/gojq"
)

// applyQuery runs a gojq expression over the JSON form of input and returns the
// resulting values. Multiple results are returned as []any.
func applyQuery(input any, expr string) (any, error) {
	if expr == "" {
		return input, nil
	}
	q, err := gojq.Parse(expr)
	if err != nil {
		return nil, fmt.Errorf("parsing --query: %w", err)
	}

	// Round-trip via JSON so structs become plain maps for jq.
	b, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	var generic any
	if err := json.Unmarshal(b, &generic); err != nil {
		return nil, err
	}

	iter := q.Run(generic)
	var out []any
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, isErr := v.(error); isErr {
			return nil, fmt.Errorf("--query: %w", err)
		}
		out = append(out, v)
	}
	if len(out) == 1 {
		return out[0], nil
	}
	return out, nil
}
```

Modify `Render` in `render.go`:
```go
func Render(w io.Writer, r Result, o Options) error {
	if o.Query != "" {
		v, err := applyQuery(unwrap(r), o.Query)
		if err != nil {
			return err
		}
		return writeRaw(w, v, o)
	}
	switch o.Format {
	case "", "json":
		return renderJSON(w, r)
	case "yaml":
		return renderYAML(w, r)
	case "ndjson":
		return renderNDJSON(w, r, o)
	case "csv":
		return renderCSV(w, r)
	case "table":
		return renderTable(w, r)
	default:
		return fmt.Errorf("unsupported output format %q", o.Format)
	}
}

func unwrap(r Result) any {
	if g, ok := r.(Get); ok {
		return g.Object
	}
	return r
}

func writeRaw(w io.Writer, v any, o Options) error {
	switch o.Format {
	case "yaml":
		enc := yaml.NewEncoder(w)
		enc.SetIndent(2)
		defer enc.Close()
		return enc.Encode(v)
	default:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	}
}
```

- [ ] **Step 4: Run test — expect PASS**

Run: `go test ./internal/output/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/output go.mod go.sum
git commit -m "output: --query gojq post-processing"
```

---

## Task 9: `output` — summary projection + `--full` + `--fields`

**Files:**
- Create: `internal/output/fields.go`
- Test: `internal/output/fields_test.go`
- Modify: `internal/output/render.go`

- [ ] **Step 1: Write the failing test**

`internal/output/fields_test.go`:
```go
package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestProjectKeepsOnlySelectedFields(t *testing.T) {
	in := map[string]any{"guid": "a", "name": "alpha", "deep": map[string]any{"k": "v"}}
	got := project(in, []string{"guid", "deep.k"})
	m := got.(map[string]any)
	if m["guid"] != "a" {
		t.Fatalf("guid lost: %v", m)
	}
	if d, ok := m["deep"].(map[string]any); !ok || d["k"] != "v" {
		t.Fatalf("deep.k lost: %v", m)
	}
	if _, has := m["name"]; has {
		t.Fatalf("name should be projected away: %v", m)
	}
}

func TestRenderListAppliesSummary(t *testing.T) {
	r := List{
		Items: []any{
			map[string]any{"guid": "a", "name": "alpha", "noise": map[string]any{"big": "data"}},
		},
		Total: 1,
	}
	var buf bytes.Buffer
	o := Options{Format: "json", SummaryFields: []string{"guid", "name"}}
	if err := Render(&buf, r, o); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "noise") {
		t.Fatalf("summary did not strip noise: %s", buf.String())
	}
}

func TestRenderListFullDisablesSummary(t *testing.T) {
	r := List{
		Items: []any{
			map[string]any{"guid": "a", "noise": "kept"},
		},
	}
	var buf bytes.Buffer
	o := Options{Format: "json", SummaryFields: []string{"guid"}, Full: true}
	if err := Render(&buf, r, o); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "noise") {
		t.Fatalf("--full should keep noise: %s", buf.String())
	}
}

func TestRenderListFieldsOverridesSummary(t *testing.T) {
	r := List{Items: []any{map[string]any{"guid": "a", "name": "alpha", "extra": "x"}}}
	var buf bytes.Buffer
	o := Options{Format: "json", SummaryFields: []string{"guid"}, Fields: []string{"guid", "extra"}}
	if err := Render(&buf, r, o); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"extra"`) {
		t.Fatalf("--fields should keep extra: %s", buf.String())
	}
	if strings.Contains(buf.String(), `"name"`) {
		t.Fatalf("--fields should drop name: %s", buf.String())
	}
}
```

- [ ] **Step 2: Run test — expect FAIL**

Run: `go test ./internal/output/... -v`
Expected: FAIL — `SummaryFields` undefined; `project` undefined.

- [ ] **Step 3: Implement**

Add to `internal/output/result.go` `Options`:
```go
type Options struct {
	Format        string
	Query         string
	Fields        []string
	Full          bool
	SummaryFields []string
	Stderr        io.Writer
}
```

`internal/output/fields.go`:
```go
package output

import (
	"encoding/json"
	"strings"
)

// effectiveFields returns the projection paths to apply, or nil for "no projection".
func effectiveFields(o Options) []string {
	switch {
	case len(o.Fields) > 0:
		return o.Fields
	case o.Full:
		return nil
	case len(o.SummaryFields) > 0:
		return o.SummaryFields
	default:
		return nil
	}
}

// project returns a copy of input keeping only the given dotted paths.
// Missing paths are silently dropped.
func project(input any, paths []string) any {
	b, err := json.Marshal(input)
	if err != nil {
		return input
	}
	var generic any
	if err := json.Unmarshal(b, &generic); err != nil {
		return input
	}
	out := map[string]any{}
	m, ok := generic.(map[string]any)
	if !ok {
		return generic
	}
	for _, p := range paths {
		copyPath(m, out, strings.Split(p, "."))
	}
	return out
}

func copyPath(src, dst map[string]any, parts []string) {
	if len(parts) == 0 {
		return
	}
	head := parts[0]
	v, ok := src[head]
	if !ok {
		return
	}
	if len(parts) == 1 {
		dst[head] = v
		return
	}
	child, ok := v.(map[string]any)
	if !ok {
		return
	}
	sub, _ := dst[head].(map[string]any)
	if sub == nil {
		sub = map[string]any{}
	}
	copyPath(child, sub, parts[1:])
	dst[head] = sub
}

func projectItems(items []any, paths []string) []any {
	if len(paths) == 0 {
		return items
	}
	out := make([]any, len(items))
	for i, it := range items {
		out[i] = project(it, paths)
	}
	return out
}
```

In `render.go` `Render`, before the format switch, apply projection:
```go
func Render(w io.Writer, r Result, o Options) error {
	r = applyProjection(r, o)
	if o.Query != "" {
		v, err := applyQuery(unwrap(r), o.Query)
		if err != nil {
			return err
		}
		return writeRaw(w, v, o)
	}
	// ... existing switch ...
}

func applyProjection(r Result, o Options) Result {
	paths := effectiveFields(o)
	if paths == nil {
		return r
	}
	switch v := r.(type) {
	case List:
		v.Items = projectItems(v.Items, paths)
		return v
	case Get:
		v.Object = project(v.Object, paths)
		return v
	default:
		return r
	}
}
```

- [ ] **Step 4: Run test — expect PASS**

Run: `go test ./internal/output/... -v`
Expected: PASS (all 4 new tests + earlier tests).

- [ ] **Step 5: Commit**

```bash
git add internal/output
git commit -m "output: summary projection, --full, and --fields"
```

---

## Task 10: `internal/safety` — audit log

**Files:**
- Create: `internal/safety/audit.go`
- Test: `internal/safety/audit_test.go`

- [ ] **Step 1: Write the failing test**

```go
package safety

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAuditAppend(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ARMOCTL_AUDIT_LOG", filepath.Join(dir, "audit.log"))

	if err := AuditAppend(Entry{
		Command:   "incidents.resolve",
		URL:       "POST /runtime/incidents/x/resolve",
		Status:    200,
		RequestID: "r1",
	}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "audit.log"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "incidents.resolve") || !strings.Contains(string(b), "requestId=r1") {
		t.Fatalf("audit line bad: %q", string(b))
	}
}
```

- [ ] **Step 2: Run test — expect FAIL**

Run: `go test ./internal/safety/... -v`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Implement**

`internal/safety/audit.go`:
```go
// Package safety wraps mutating CLI commands with dry-run, confirmation,
// and audit logging.
package safety

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Entry is a single audit log record.
type Entry struct {
	Command   string
	URL       string
	Status    int
	RequestID string
	Args      string // already-redacted single-line args
}

// AuditAppend writes one line to the audit log.
// Path: $ARMOCTL_AUDIT_LOG, else ~/.armoctl/audit.log.
func AuditAppend(e Entry) error {
	path := os.Getenv("ARMOCTL_AUDIT_LOG")
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		path = filepath.Join(home, ".armoctl", "audit.log")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	line := fmt.Sprintf("%s %s args=%q url=%s status=%d requestId=%s\n",
		time.Now().UTC().Format(time.RFC3339), e.Command, e.Args, e.URL, e.Status, e.RequestID)
	_, err = f.WriteString(line)
	return err
}
```

- [ ] **Step 4: Run test — expect PASS**

Run: `go test ./internal/safety/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/safety
git commit -m "safety: audit log append helper"
```

---

## Task 11: `safety` — Wrap mutation: dry-run, --yes, tty/non-tty

**Files:**
- Create: `internal/safety/safety.go`
- Test: `internal/safety/safety_test.go`

- [ ] **Step 1: Write the failing test**

```go
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
```

- [ ] **Step 2: Run test — expect FAIL**

Run: `go test ./internal/safety/... -v`
Expected: FAIL — `Wrap`/`Args`/`ExecMeta` undefined.

- [ ] **Step 3: Implement**

`internal/safety/safety.go`:
```go
package safety

import (
	"context"
	"encoding/json"
	"io"

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
	if a.Stdout == nil {
		return &clierr.Error{Code: clierr.CodeBadInput, Msg: "safety.Wrap: missing Stdout"}
	}

	if a.DryRun {
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
		// TTY confirmation is performed by the caller before reaching here in tests;
		// production callers use askConfirm. Keep it simple: refuse if not confirmed.
		ok, err := askConfirm(a.Stderr)
		if err != nil {
			return err
		}
		if !ok {
			return &clierr.Error{Code: clierr.CodeNeedsConfirm, Msg: "user declined"}
		}
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
```

Add stub `askConfirm` in same file:
```go
import "bufio"
import "strings"

func askConfirm(w io.Writer) (bool, error) {
	if w == nil {
		return false, nil
	}
	io.WriteString(w, "proceed? [y/N] ")
	br := bufio.NewReader(stdinReader())
	line, err := br.ReadString('\n')
	if err != nil {
		return false, nil
	}
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes", nil
}
```

And a separable `stdinReader()` for tests:
```go
import "os"

var stdinReader = func() io.Reader { return os.Stdin }
```

- [ ] **Step 4: Run test — expect PASS**

Run: `go test ./internal/safety/... -v`
Expected: PASS (all 4 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/safety
git commit -m "safety: Wrap mutation with dry-run, --yes, tty handling"
```

---

## Task 12: `internal/schema` — embed schemas + `armoctl schema` command

**Files:**
- Create: `internal/schema/schema.go`
- Create: `internal/schema/data/.gitkeep`
- Create: `internal/schema/data/incidents.json`
- Test: `internal/schema/schema_test.go`

- [ ] **Step 1: Write the failing test**

```go
package schema

import (
	"strings"
	"testing"
)

func TestListEnumeratesEmbeddedResources(t *testing.T) {
	names := List()
	found := false
	for _, n := range names {
		if n == "incidents" {
			found = true
		}
	}
	if !found {
		t.Fatalf("incidents not in List(): %v", names)
	}
}

func TestGetReturnsJSONSchemaBytes(t *testing.T) {
	b, err := Get("incidents")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"$schema"`) && !strings.Contains(string(b), `"type"`) {
		t.Fatalf("schema content missing JSON schema markers: %s", string(b))
	}
}
```

- [ ] **Step 2: Run test — expect FAIL**

Run: `go test ./internal/schema/... -v`
Expected: FAIL — package not implemented.

- [ ] **Step 3: Implement**

`internal/schema/data/incidents.json` (minimal placeholder, generator script in Task 13 will overwrite it):
```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "Incident",
  "type": "object",
  "properties": {
    "guid": {"type": "string"},
    "name": {"type": "string"},
    "severity": {"type": "string"},
    "status": {"type": "string"},
    "alerts": {"type": "array", "items": {"type": "object"}}
  }
}
```

`internal/schema/data/.gitkeep`: empty file.

`internal/schema/schema.go`:
```go
// Package schema embeds JSON schemas for armoctl resources and provides
// the `armoctl schema` cobra command.
package schema

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/armosec/armoctl/internal/clierr"
	"github.com/spf13/cobra"
)

//go:embed data/*.json
var fsys embed.FS

// List returns the resource names whose schemas are embedded.
func List() []string {
	var names []string
	_ = fs.WalkDir(fsys, "data", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		names = append(names, strings.TrimSuffix(strings.TrimPrefix(p, "data/"), ".json"))
		return nil
	})
	sort.Strings(names)
	return names
}

// Get returns the JSON schema bytes for resource.
func Get(resource string) ([]byte, error) {
	b, err := fsys.ReadFile("data/" + resource + ".json")
	if err != nil {
		return nil, &clierr.Error{Code: clierr.CodeNotFound, Msg: fmt.Sprintf("no schema for %q", resource)}
	}
	// validate parseable
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, fmt.Errorf("schema for %s is not valid JSON: %w", resource, err)
	}
	return b, nil
}

// Cmd builds the `armoctl schema` cobra command.
func Cmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "schema [resource]",
		Short: "Print JSON schemas for armoctl resources",
		RunE: func(cmd *cobra.Command, args []string) error {
			list, _ := cmd.Flags().GetBool("list")
			if list {
				for _, n := range List() {
					fmt.Fprintln(cmd.OutOrStdout(), n)
				}
				return nil
			}
			if len(args) != 1 {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "schema requires a resource name (or --list)"}
			}
			b, err := Get(args[0])
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(b))
			return nil
		},
	}
	c.Flags().Bool("list", false, "List embedded schema resource names")
	return c
}
```

- [ ] **Step 4: Run test — expect PASS**

Run: `go test ./internal/schema/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/schema
git commit -m "schema: embedded JSON schemas + armoctl schema command"
```

---

## Task 13: `scripts/gen-schemas.sh` — schema generator skeleton

**Files:**
- Create: `scripts/gen-schemas.sh`
- Modify: `Makefile` (add `schemas` target)

- [ ] **Step 1: Write the script (no test — generation correctness is checked by the embedding tests in Task 12)**

`scripts/gen-schemas.sh`:
```bash
#!/usr/bin/env bash
# Regenerates JSON schemas under internal/schema/data/ from
# cadashboardbe's docs/swagger.json and from armotypes Go struct tags.
#
# Inputs:
#   $SWAGGER_PATH - path to swagger.json (default: ../cadashboardbe/docs/swagger.json)
# Outputs:
#   internal/schema/data/*.json (one per resource)
#
# This is the v1 generator: it copies a hand-curated allowlist of definitions
# from swagger.json. Per-cluster plans add resources to RESOURCES below.

set -euo pipefail

SWAGGER_PATH="${SWAGGER_PATH:-../cadashboardbe/docs/swagger.json}"
OUT_DIR="$(dirname "$0")/../internal/schema/data"

# resource:swaggerDefinitionName
RESOURCES=(
  "incidents:RuntimeIncident"
)

if [[ ! -f "$SWAGGER_PATH" ]]; then
  echo "swagger not found at $SWAGGER_PATH" >&2
  exit 2
fi

for entry in "${RESOURCES[@]}"; do
  name="${entry%%:*}"
  defn="${entry##*:}"
  out="$OUT_DIR/$name.json"
  jq --arg d "$defn" '.definitions[$d] // (.components.schemas[$d] // null)' "$SWAGGER_PATH" \
    | jq '. + {"$schema":"https://json-schema.org/draft/2020-12/schema"}' \
    > "$out.tmp"
  if ! jq -e 'type=="object"' "$out.tmp" >/dev/null; then
    echo "definition $defn not found in swagger" >&2
    rm -f "$out.tmp"
    exit 3
  fi
  mv "$out.tmp" "$out"
  echo "wrote $out"
done
```

`chmod +x scripts/gen-schemas.sh`.

- [ ] **Step 2: Add Makefile target**

Append to existing `Makefile` (or create one if absent):
```make
.PHONY: schemas
schemas:
	./scripts/gen-schemas.sh
```

- [ ] **Step 3: Smoke-run with the real swagger**

Run: `make schemas` (must be run from a tree where `../cadashboardbe/docs/swagger.json` exists, or pass `SWAGGER_PATH=`).
Expected: `internal/schema/data/incidents.json` is regenerated. If the definition name in swagger differs (e.g. `RuntimeAlertIncident`), update the `RESOURCES` list and re-run.
If the swagger path isn't available in this environment, skip this step and fix the resource name when first wiring CI.

- [ ] **Step 4: Run schema tests to confirm regenerated data still parses**

Run: `go test ./internal/schema/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add scripts/gen-schemas.sh Makefile internal/schema/data/incidents.json
git commit -m "schema: gen-schemas.sh skeleton and Makefile target"
```

---

## Task 14: `cmd/cliflags` — register shared persistent flags + value getter

**Files:**
- Create: `cmd/cliflags/flags.go`
- Test: `cmd/cliflags/flags_test.go`

- [ ] **Step 1: Write the failing test**

```go
package cliflags

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestRegisterAddsExpectedFlags(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	Register(root)
	for _, name := range []string{"output", "query", "fields", "full", "limit", "page", "page-size", "dry-run", "yes"} {
		if root.PersistentFlags().Lookup(name) == nil {
			t.Errorf("flag %q not registered", name)
		}
	}
}
```

- [ ] **Step 2: Run test — expect FAIL**

Run: `go test ./cmd/cliflags/... -v`
Expected: FAIL — package not present.

- [ ] **Step 3: Implement**

`cmd/cliflags/flags.go`:
```go
// Package cliflags registers the persistent flags shared across every
// armoctl resource command and exposes a single Resolve() to read them.
package cliflags

import (
	"strings"

	"github.com/armosec/armoctl/internal/output"
	"github.com/spf13/cobra"
)

// Register installs the persistent flags on root.
func Register(root *cobra.Command) {
	pf := root.PersistentFlags()
	pf.String("output", "json", "Output format: json|yaml|ndjson|table|csv")
	pf.String("query", "", "gojq expression applied after --fields/summary")
	pf.StringSlice("fields", nil, "Comma-separated dotted paths to keep")
	pf.Bool("full", false, "Disable summary projection; return raw response")
	pf.Int("limit", 500, "Max items to fetch when auto-paging (0 = no cap, requires --yes)")
	pf.Int("page", 0, "Explicit page (1-based; disables auto-paging)")
	pf.Int("page-size", 0, "Explicit page size (disables auto-paging)")
	pf.Bool("dry-run", false, "Build the request but do not send it")
	pf.Bool("yes", false, "Skip confirmation for mutations")
}

// OutputOptions reads the output-related flags from cmd.
func OutputOptions(cmd *cobra.Command, summary []string) output.Options {
	o := output.Options{
		Format:        flagString(cmd, "output"),
		Query:         flagString(cmd, "query"),
		Full:          flagBool(cmd, "full"),
		SummaryFields: summary,
	}
	if raw := flagString(cmd, "fields"); raw != "" {
		o.Fields = splitFields(raw)
	} else if fs, _ := cmd.Flags().GetStringSlice("fields"); len(fs) > 0 {
		o.Fields = fs
	}
	return o
}

func splitFields(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// PageOptions reads the pagination flags.
type PageOptions struct {
	Limit    int
	Page     int
	PageSize int
}

func ReadPage(cmd *cobra.Command) PageOptions {
	return PageOptions{
		Limit:    flagInt(cmd, "limit"),
		Page:     flagInt(cmd, "page"),
		PageSize: flagInt(cmd, "page-size"),
	}
}

// MutationOptions reads dry-run / yes.
type MutationOptions struct {
	DryRun bool
	Yes    bool
}

func ReadMutation(cmd *cobra.Command) MutationOptions {
	return MutationOptions{DryRun: flagBool(cmd, "dry-run"), Yes: flagBool(cmd, "yes")}
}

func flagString(cmd *cobra.Command, n string) string {
	v, _ := cmd.Flags().GetString(n)
	return v
}
func flagBool(cmd *cobra.Command, n string) bool { v, _ := cmd.Flags().GetBool(n); return v }
func flagInt(cmd *cobra.Command, n string) int   { v, _ := cmd.Flags().GetInt(n); return v }
```

- [ ] **Step 4: Run test — expect PASS**

Run: `go test ./cmd/cliflags/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/cliflags
git commit -m "cliflags: shared persistent flags and option readers"
```

---

## Task 15: `cmd/incidents` — types + cluster root + `fields` cheatsheet

**Files:**
- Create: `cmd/incidents/incidents.go`
- Create: `cmd/incidents/types.go`
- Create: `cmd/incidents/fields.go`
- Test: `cmd/incidents/fields_test.go`

- [ ] **Step 1: Write the failing test**

`cmd/incidents/fields_test.go`:
```go
package incidents

import (
	"bytes"
	"strings"
	"testing"
)

func TestFieldsCheatsheetIsNonEmpty(t *testing.T) {
	cheatsheet := Cheatsheet()
	if len(cheatsheet) < 5 {
		t.Fatalf("cheatsheet too small: %d", len(cheatsheet))
	}
	have := map[string]bool{}
	for _, f := range cheatsheet {
		have[f.Name] = true
	}
	for _, want := range []string{"guid", "name", "severity"} {
		if !have[want] {
			t.Errorf("cheatsheet missing %q", want)
		}
	}
}

func TestFieldsCommandPrintsCheatsheet(t *testing.T) {
	cmd := FieldsCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs(nil)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "guid") {
		t.Fatalf("output missing guid: %q", buf.String())
	}
}
```

- [ ] **Step 2: Run test — expect FAIL**

Run: `go test ./cmd/incidents/... -v`
Expected: FAIL — package missing.

- [ ] **Step 3: Implement**

`cmd/incidents/types.go`:
```go
// Package incidents implements the `armoctl incidents` cluster.
package incidents

// SummaryFields is the default projection applied to `incidents list`.
var SummaryFields = []string{
	"guid", "name", "severity", "status", "creationTimestamp",
	"resource.cluster", "resource.namespace", "resource.workload",
}

// Field is one entry in the per-resource cheatsheet.
type Field struct {
	Name string
	Doc  string
}

// Cheatsheet returns the curated field list used both for `armoctl incidents fields`
// and for the auto-generated section in SKILL.md.
func Cheatsheet() []Field {
	return []Field{
		{"guid", "Stable incident ID; primary key for get/resolve/explain."},
		{"name", "Short rule/incident name (e.g. \"Suspicious binary\")."},
		{"severity", "critical | high | medium | low."},
		{"status", "open | resolved | investigating."},
		{"creationTimestamp", "RFC3339 time the incident was raised."},
		{"resource.cluster", "Cluster the workload belongs to."},
		{"resource.namespace", "Kubernetes namespace (or N/A for ECS)."},
		{"resource.workload", "Workload name (deployment/service/task)."},
		{"alertCount", "Number of alerts grouped under this incident."},
		{"resolvedBy", "User/service that resolved the incident, if any."},
		{"resolutionReason", "Free-text reason recorded at resolve time."},
	}
}
```

`cmd/incidents/incidents.go`:
```go
package incidents

import "github.com/spf13/cobra"

// Cmd builds the `armoctl incidents` cluster root.
func Cmd() *cobra.Command {
	c := &cobra.Command{Use: "incidents", Short: "Inspect and manage runtime incidents"}
	c.AddCommand(FieldsCmd())
	// list/get/alerts/explain/resolve/unresolve/severities are added in subsequent tasks.
	return c
}
```

`cmd/incidents/fields.go`:
```go
package incidents

import (
	"fmt"

	"github.com/spf13/cobra"
)

// FieldsCmd is `armoctl incidents fields`.
func FieldsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fields",
		Short: "Print the incidents resource cheatsheet",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "Default summary fields (use --full or --fields to override):")
			for _, f := range SummaryFields {
				fmt.Fprintf(out, "  %s\n", f)
			}
			fmt.Fprintln(out)
			fmt.Fprintln(out, "Field cheatsheet:")
			for _, f := range Cheatsheet() {
				fmt.Fprintf(out, "  %-22s %s\n", f.Name, f.Doc)
			}
			return nil
		},
	}
}
```

- [ ] **Step 4: Run test — expect PASS**

Run: `go test ./cmd/incidents/... -v`
Expected: PASS (both tests).

- [ ] **Step 5: Commit**

```bash
git add cmd/incidents
git commit -m "incidents: types, cluster root, and fields cheatsheet"
```

---

## Task 16: `cmd/incidents` — `list` (with auto-paging in apiclient)

**Files:**
- Modify: `internal/apiclient/paging.go` (new)
- Test: `internal/apiclient/paging_test.go`
- Create: `cmd/incidents/list.go`
- Test: `cmd/incidents/list_test.go`
- Modify: `cmd/incidents/incidents.go`

- [ ] **Step 1: Write the failing test (apiclient paging)**

`internal/apiclient/paging_test.go`:
```go
package apiclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
)

func TestListPaged_AutoPagesUntilLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("pageNum"))
		size, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
		if size == 0 {
			size = 50
		}
		total := 7
		start := page * size
		end := start + size
		if end > total {
			end = total
		}
		items := []map[string]any{}
		for i := start; i < end; i++ {
			items = append(items, map[string]any{"i": i})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": items,
			"total":    map[string]any{"value": total},
		})
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	got, err := c.ListPaged(context.Background(), "/runtime/incidents", url.Values{}, ListOpts{Limit: 5, PageSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	if got.Total != 7 {
		t.Fatalf("total = %d, want 7", got.Total)
	}
	if len(got.Items) != 5 {
		t.Fatalf("items = %d, want 5 (capped by Limit)", len(got.Items))
	}
	if got.Items[0].(map[string]any)["i"].(float64) != 0 || got.Items[4].(map[string]any)["i"].(float64) != 4 {
		t.Fatalf("items not in order: %v", got.Items)
	}
	_ = fmt.Sprintf
}
```

- [ ] **Step 2: Run test — expect FAIL**

Run: `go test ./internal/apiclient/... -v`
Expected: FAIL — `ListPaged`/`ListOpts` undefined.

- [ ] **Step 3: Implement**

`internal/apiclient/paging.go`:
```go
package apiclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// ListOpts controls paged list calls.
type ListOpts struct {
	Limit    int // total cap on items collected (0 = no cap)
	Page     int // explicit page (1-based) — disables auto-paging when >0
	PageSize int // page size to request (default 50)
}

// PagedResult is the apiclient's normalized paged response.
type PagedResult struct {
	Items      []any
	Total      int
	Page       int
	PageSize   int
	NextCursor string
}

type rawListResponse struct {
	Response   []json.RawMessage `json:"response"`
	Total      *struct {
		Value int `json:"value"`
	} `json:"total"`
	NextCursor string `json:"nextCursor"`
}

// ListPaged executes a paged GET. The ARMO API uses `pageNum` (0-based) and
// `pageSize`. Auto-pages until len(items) >= opts.Limit, opts.Limit == 0 reached,
// or the server reports fewer items than pageSize.
func (c *Client) ListPaged(ctx context.Context, path string, query url.Values, opts ListOpts) (PagedResult, error) {
	if opts.PageSize <= 0 {
		opts.PageSize = 50
	}
	if opts.Page > 0 {
		// Explicit single page mode.
		q := cloneValues(query)
		q.Set("pageNum", strconv.Itoa(opts.Page-1))
		q.Set("pageSize", strconv.Itoa(opts.PageSize))
		raw, err := c.fetchPage(ctx, path, q)
		if err != nil {
			return PagedResult{}, err
		}
		items, err := unwrapItems(raw.Response)
		if err != nil {
			return PagedResult{}, err
		}
		total := 0
		if raw.Total != nil {
			total = raw.Total.Value
		}
		return PagedResult{Items: items, Total: total, Page: opts.Page, PageSize: opts.PageSize, NextCursor: raw.NextCursor}, nil
	}

	page := 0
	out := PagedResult{Page: 1, PageSize: opts.PageSize}
	for {
		q := cloneValues(query)
		q.Set("pageNum", strconv.Itoa(page))
		q.Set("pageSize", strconv.Itoa(opts.PageSize))
		raw, err := c.fetchPage(ctx, path, q)
		if err != nil {
			return PagedResult{}, err
		}
		items, err := unwrapItems(raw.Response)
		if err != nil {
			return PagedResult{}, err
		}
		out.Items = append(out.Items, items...)
		if raw.Total != nil {
			out.Total = raw.Total.Value
		}
		out.NextCursor = raw.NextCursor

		if opts.Limit > 0 && len(out.Items) >= opts.Limit {
			out.Items = out.Items[:opts.Limit]
			return out, nil
		}
		if len(items) < opts.PageSize {
			return out, nil
		}
		page++
	}
}

func (c *Client) fetchPage(ctx context.Context, path string, q url.Values) (rawListResponse, error) {
	resp, err := c.Do(ctx, http.MethodGet, path, q, nil)
	if err != nil {
		return rawListResponse{}, err
	}
	defer resp.Body.Close()
	var raw rawListResponse
	if err := decode(resp, &raw); err != nil {
		return rawListResponse{}, err
	}
	return raw, nil
}

func unwrapItems(rows []json.RawMessage) ([]any, error) {
	out := make([]any, 0, len(rows))
	for _, r := range rows {
		var v any
		if err := json.Unmarshal(r, &v); err != nil {
			return nil, fmt.Errorf("decoding list item: %w", err)
		}
		out = append(out, v)
	}
	return out, nil
}

func cloneValues(v url.Values) url.Values {
	out := url.Values{}
	for k, vs := range v {
		for _, vv := range vs {
			out.Add(k, vv)
		}
	}
	return out
}
```

- [ ] **Step 4: Run test — expect PASS**

Run: `go test ./internal/apiclient/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/apiclient
git commit -m "apiclient: ListPaged with auto-paging and explicit-page modes"
```

---

## Task 17: `cmd/incidents list` — wire it end-to-end

**Files:**
- Create: `cmd/incidents/list.go`
- Test: `cmd/incidents/list_test.go`
- Modify: `cmd/incidents/incidents.go`

- [ ] **Step 1: Write the failing test**

`cmd/incidents/list_test.go`:
```go
package incidents

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/spf13/cobra"
)

func TestList_PrintsItemsAsJSONList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": []map[string]any{
				{"guid": "i1", "name": "Suspicious binary", "severity": "high", "status": "open", "noise": "x"},
				{"guid": "i2", "name": "C2 beacon", "severity": "critical", "status": "open", "noise": "y"},
			},
			"total": map[string]any{"value": 2},
		})
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})

	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	root.AddCommand(ListCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))

	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stdout)
	root.SetArgs([]string{"list"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	out := stdout.String()
	if !strings.Contains(out, `"items"`) || !strings.Contains(out, "i1") || !strings.Contains(out, "i2") {
		t.Fatalf("unexpected list output: %s", out)
	}
	if strings.Contains(out, "noise") {
		t.Fatalf("default summary should drop noise: %s", out)
	}
}
```

- [ ] **Step 2: Run test — expect FAIL**

Run: `go test ./cmd/incidents/... -v`
Expected: FAIL — `ListCmd` undefined.

- [ ] **Step 3: Implement**

`cmd/incidents/list.go`:
```go
package incidents

import (
	"net/url"

	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/armosec/armoctl/internal/output"
	"github.com/spf13/cobra"
)

// ClientFor returns the apiclient configured for the running command.
// Cluster commands take this as a function so tests can inject stubs.
type ClientFor func(cmd *cobra.Command) *apiclient.Client

// ListCmd builds `armoctl incidents list`.
func ListCmd(clientFor ClientFor) *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "List runtime incidents",
		RunE: func(cmd *cobra.Command, args []string) error {
			cli := clientFor(cmd)
			pg := cliflags.ReadPage(cmd)
			q := url.Values{}
			if sev, _ := cmd.Flags().GetString("severity"); sev != "" {
				q.Set("severity", sev)
			}
			res, err := cli.ListPaged(cmd.Context(), "/runtime/incidents", q, apiclient.ListOpts{
				Limit: pg.Limit, Page: pg.Page, PageSize: pg.PageSize,
			})
			if err != nil {
				return err
			}
			list := output.List{
				Items: res.Items, Total: res.Total,
				Page: res.Page, PageSize: res.PageSize, NextCursor: res.NextCursor,
			}
			return output.Render(cmd.OutOrStdout(), list, cliflags.OutputOptions(cmd, SummaryFields))
		},
	}
	c.Flags().String("severity", "", "Filter by severity (critical|high|medium|low)")
	return c
}
```

Modify `cmd/incidents/incidents.go`:
```go
package incidents

import (
	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/spf13/cobra"
)

func Cmd(clientFor ClientFor) *cobra.Command {
	c := &cobra.Command{Use: "incidents", Short: "Inspect and manage runtime incidents"}
	c.AddCommand(FieldsCmd())
	c.AddCommand(ListCmd(clientFor))
	return c
}

// DefaultClientFor reads viper config and builds an apiclient.
// It's exported so root.go can pass it without coupling cobra to apiclient.
func DefaultClientFor(read func(key string) string) ClientFor {
	return func(cmd *cobra.Command) *apiclient.Client {
		return apiclient.New(apiclient.Config{
			BaseURL:      "https://" + read("api-url"),
			AccessKey:    read("access-key"),
			CustomerGUID: read("customer-guid"),
		})
	}
}
```

- [ ] **Step 4: Run test — expect PASS**

Run: `go test ./cmd/incidents/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/incidents
git commit -m "incidents: list with default summary projection"
```

---

## Task 18: `cmd/incidents get`

**Files:**
- Create: `cmd/incidents/get.go`
- Test: `cmd/incidents/get_test.go`
- Modify: `cmd/incidents/incidents.go`

- [ ] **Step 1: Write the failing test**

```go
package incidents

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/spf13/cobra"
)

func TestGet_PrintsObject(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/runtime/incidents/i1") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"guid": "i1", "name": "X"})
	}))
	defer srv.Close()
	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})

	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	root.AddCommand(GetCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))

	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"get", "i1", "--full"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), `"guid": "i1"`) {
		t.Fatalf("output: %s", stdout.String())
	}
}
```

- [ ] **Step 2: Run test — expect FAIL**

Run: `go test ./cmd/incidents/... -v`
Expected: FAIL — `GetCmd` undefined.

- [ ] **Step 3: Implement**

`cmd/incidents/get.go`:
```go
package incidents

import (
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/armosec/armoctl/internal/output"
	"github.com/spf13/cobra"
)

func GetCmd(clientFor ClientFor) *cobra.Command {
	return &cobra.Command{
		Use:   "get [guid]",
		Short: "Get a single incident by GUID",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "get requires a GUID"}
			}
			cli := clientFor(cmd)
			var obj map[string]any
			if err := cli.GetJSON(cmd.Context(), "/runtime/incidents/"+args[0], nil, &obj); err != nil {
				return err
			}
			return output.Render(cmd.OutOrStdout(), output.Get{Object: obj}, cliflags.OutputOptions(cmd, SummaryFields))
		},
	}
}
```

In `incidents.go` add `c.AddCommand(GetCmd(clientFor))`.

- [ ] **Step 4: Run test — expect PASS**

Run: `go test ./cmd/incidents/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/incidents
git commit -m "incidents: get subcommand"
```

---

## Task 19: `cmd/incidents alerts`

**Files:**
- Create: `cmd/incidents/alerts.go`
- Test: `cmd/incidents/alerts_test.go`
- Modify: `cmd/incidents/incidents.go`

- [ ] **Step 1: Write the failing test**

```go
package incidents

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/spf13/cobra"
)

func TestAlerts_ListsAlertsForIncident(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/runtime/incidents/i1/alerts/list") {
			t.Errorf("path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": []map[string]any{{"alertID": "a1"}},
			"total":    map[string]any{"value": 1},
		})
	}))
	defer srv.Close()
	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})

	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	root.AddCommand(AlertsCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))

	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"alerts", "i1", "--full"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "a1") {
		t.Fatalf("output: %s", stdout.String())
	}
}
```

- [ ] **Step 2: Run test — expect FAIL**

Run: `go test ./cmd/incidents/... -v`
Expected: FAIL.

- [ ] **Step 3: Implement**

`cmd/incidents/alerts.go`:
```go
package incidents

import (
	"net/url"

	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/armosec/armoctl/internal/output"
	"github.com/spf13/cobra"
)

func AlertsCmd(clientFor ClientFor) *cobra.Command {
	return &cobra.Command{
		Use:   "alerts [guid]",
		Short: "List alerts grouped under one incident",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "alerts requires an incident GUID"}
			}
			cli := clientFor(cmd)
			pg := cliflags.ReadPage(cmd)
			path := "/runtime/incidents/" + args[0] + "/alerts/list"
			res, err := cli.ListPaged(cmd.Context(), path, url.Values{}, apiclient.ListOpts{
				Limit: pg.Limit, Page: pg.Page, PageSize: pg.PageSize,
			})
			if err != nil {
				return err
			}
			list := output.List{Items: res.Items, Total: res.Total, Page: res.Page, PageSize: res.PageSize, NextCursor: res.NextCursor}
			return output.Render(cmd.OutOrStdout(), list, cliflags.OutputOptions(cmd, nil))
		},
	}
}
```

Add to `incidents.go`: `c.AddCommand(AlertsCmd(clientFor))`.

- [ ] **Step 4: Run test — expect PASS**

Run: `go test ./cmd/incidents/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/incidents
git commit -m "incidents: alerts subcommand"
```

---

## Task 20: `cmd/incidents explain`

**Files:**
- Create: `cmd/incidents/explain.go`
- Test: `cmd/incidents/explain_test.go`
- Modify: `cmd/incidents/incidents.go`

- [ ] **Step 1: Write the failing test**

```go
package incidents

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/spf13/cobra"
)

func TestExplain_PrintsExplanation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"explanation": "process spawned shell"})
	}))
	defer srv.Close()
	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})

	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	root.AddCommand(ExplainCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))

	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"explain", "i1", "--full"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "spawned shell") {
		t.Fatalf("output: %s", stdout.String())
	}
}
```

- [ ] **Step 2: Run test — expect FAIL**

Run: `go test ./cmd/incidents/... -v`
Expected: FAIL.

- [ ] **Step 3: Implement**

`cmd/incidents/explain.go`:
```go
package incidents

import (
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/armosec/armoctl/internal/output"
	"github.com/spf13/cobra"
)

func ExplainCmd(clientFor ClientFor) *cobra.Command {
	return &cobra.Command{
		Use:   "explain [guid]",
		Short: "Get the platform's explanation for an incident",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "explain requires a GUID"}
			}
			cli := clientFor(cmd)
			var obj map[string]any
			if err := cli.GetJSON(cmd.Context(), "/runtime/incidents/"+args[0]+"/explain", nil, &obj); err != nil {
				return err
			}
			return output.Render(cmd.OutOrStdout(), output.Get{Object: obj}, cliflags.OutputOptions(cmd, nil))
		},
	}
}
```

Add `c.AddCommand(ExplainCmd(clientFor))` in `incidents.go`.

- [ ] **Step 4: Run test — expect PASS**

Run: `go test ./cmd/incidents/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/incidents
git commit -m "incidents: explain subcommand"
```

---

## Task 21: `cmd/incidents resolve` (mutation, safety wrap)

**Files:**
- Create: `cmd/incidents/resolve.go`
- Test: `cmd/incidents/resolve_test.go`
- Modify: `cmd/incidents/incidents.go`

- [ ] **Step 1: Write the failing test**

```go
package incidents

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/spf13/cobra"
)

func TestResolve_DryRunDoesNotCallServer(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(200)
		_, _ = w.Write([]byte("{}"))
	}))
	defer srv.Close()
	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})

	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	root.AddCommand(ResolveCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))

	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"resolve", "i1", "--reason", "fp", "--dry-run"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&hits) != 0 {
		t.Fatalf("server was called during dry-run")
	}
	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("output not JSON: %v: %q", err, stdout.String())
	}
	if got["dryRun"] != true {
		t.Fatalf("dryRun: %v", got)
	}
}

func TestResolve_YesPostsAndReportsChanged(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method: %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/runtime/incidents/i1/resolve") {
			t.Errorf("path: %s", r.URL.Path)
		}
		w.Header().Set("x-request-id", "req-z")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"resolved":true}`))
	}))
	defer srv.Close()
	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})

	t.Setenv("ARMOCTL_AUDIT_LOG", t.TempDir()+"/audit.log")

	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	root.AddCommand(ResolveCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))

	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"resolve", "i1", "--reason", "fp", "--yes"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), `"changed": true`) {
		t.Fatalf("changed not set: %s", stdout.String())
	}
}
```

- [ ] **Step 2: Run test — expect FAIL**

Run: `go test ./cmd/incidents/... -v`
Expected: FAIL — `ResolveCmd` undefined.

- [ ] **Step 3: Implement**

`cmd/incidents/resolve.go`:
```go
package incidents

import (
	"context"

	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/armosec/armoctl/internal/safety"
	"github.com/spf13/cobra"
)

func ResolveCmd(clientFor ClientFor) *cobra.Command {
	c := &cobra.Command{
		Use:   "resolve [guid]",
		Short: "Resolve a runtime incident",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "resolve requires a GUID"}
			}
			reason, _ := cmd.Flags().GetString("reason")
			body := map[string]any{"reason": reason}
			path := "/runtime/incidents/" + args[0] + "/resolve"

			cli := clientFor(cmd)
			m := cliflags.ReadMutation(cmd)

			return safety.Wrap(cmd.Context(), safety.Args{
				Command: "incidents.resolve",
				DryRun:  m.DryRun,
				Yes:     m.Yes,
				Tty:     false,
				Stdout:  cmd.OutOrStdout(),
				Stderr:  cmd.ErrOrStderr(),
				Preview: map[string]any{"method": "POST", "url": path, "body": body},
				ArgsLog: "guid=" + args[0],
				Exec: func(ctx context.Context) (any, safety.ExecMeta, error) {
					var resp map[string]any
					err := cli.PostJSON(ctx, path, nil, body, &resp)
					if err != nil {
						return nil, safety.ExecMeta{}, err
					}
					return resp, safety.ExecMeta{URL: "POST " + path, Status: 200}, nil
				},
			})
		},
	}
	c.Flags().String("reason", "", "Free-text reason recorded with the resolution")
	_ = c.MarkFlagRequired("reason")
	_ = apiclient.Config{} // keep import used in tests
	return c
}
```

Add `c.AddCommand(ResolveCmd(clientFor))` in `incidents.go`.

- [ ] **Step 4: Run test — expect PASS**

Run: `go test ./cmd/incidents/... -v`
Expected: PASS (both tests).

- [ ] **Step 5: Commit**

```bash
git add cmd/incidents
git commit -m "incidents: resolve subcommand with safety wrap and audit"
```

---

## Task 22: `cmd/incidents unresolve` and `severities`

**Files:**
- Create: `cmd/incidents/unresolve.go`
- Create: `cmd/incidents/severities.go`
- Test: `cmd/incidents/severities_test.go`
- Modify: `cmd/incidents/incidents.go`

- [ ] **Step 1: Write the failing test**

```go
package incidents

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/spf13/cobra"
)

func TestSeverities_ReturnsAggregate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"critical":3,"high":7,"medium":1,"low":0}`))
	}))
	defer srv.Close()
	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})

	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	root.AddCommand(SeveritiesCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))

	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"severities", "--full"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), `"high": 7`) {
		t.Fatalf("output: %s", stdout.String())
	}
}
```

- [ ] **Step 2: Run test — expect FAIL**

Run: `go test ./cmd/incidents/... -v`
Expected: FAIL — `SeveritiesCmd` undefined.

- [ ] **Step 3: Implement**

`cmd/incidents/severities.go`:
```go
package incidents

import (
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/output"
	"github.com/spf13/cobra"
)

func SeveritiesCmd(clientFor ClientFor) *cobra.Command {
	return &cobra.Command{
		Use:   "severities",
		Short: "Get aggregate incident counts per severity",
		RunE: func(cmd *cobra.Command, args []string) error {
			cli := clientFor(cmd)
			var obj map[string]any
			if err := cli.GetJSON(cmd.Context(), "/runtime/incidentsPerSeverity", nil, &obj); err != nil {
				return err
			}
			return output.Render(cmd.OutOrStdout(), output.Get{Object: obj}, cliflags.OutputOptions(cmd, nil))
		},
	}
}
```

`cmd/incidents/unresolve.go`:
```go
package incidents

import (
	"context"

	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/armosec/armoctl/internal/safety"
	"github.com/spf13/cobra"
)

func UnresolveCmd(clientFor ClientFor) *cobra.Command {
	return &cobra.Command{
		Use:   "unresolve [guid]",
		Short: "Reopen a previously-resolved incident",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "unresolve requires a GUID"}
			}
			path := "/runtime/incidents/" + args[0] + "/unresolve"
			cli := clientFor(cmd)
			m := cliflags.ReadMutation(cmd)
			return safety.Wrap(cmd.Context(), safety.Args{
				Command: "incidents.unresolve",
				DryRun:  m.DryRun,
				Yes:     m.Yes,
				Tty:     false,
				Stdout:  cmd.OutOrStdout(),
				Stderr:  cmd.ErrOrStderr(),
				Preview: map[string]any{"method": "POST", "url": path},
				ArgsLog: "guid=" + args[0],
				Exec: func(ctx context.Context) (any, safety.ExecMeta, error) {
					var resp map[string]any
					if err := cli.PostJSON(ctx, path, nil, nil, &resp); err != nil {
						return nil, safety.ExecMeta{}, err
					}
					return resp, safety.ExecMeta{URL: "POST " + path, Status: 200}, nil
				},
			})
		},
	}
}
```

In `incidents.go`:
```go
c.AddCommand(UnresolveCmd(clientFor))
c.AddCommand(SeveritiesCmd(clientFor))
```

- [ ] **Step 4: Run test — expect PASS**

Run: `go test ./cmd/incidents/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/incidents
git commit -m "incidents: unresolve and severities subcommands"
```

---

## Task 23: Wire `incidents` and `schema` clusters into root + main

**Files:**
- Modify: `root.go`

- [ ] **Step 1: Edit root.go**

Replace `init()` body in `root.go` with the additions below; keep existing ECS wiring.

```go
import (
	// existing imports ...
	"github.com/armosec/armoctl/cmd/cliflags"
	incidentscmd "github.com/armosec/armoctl/cmd/incidents"
	schemacmd "github.com/armosec/armoctl/internal/schema"
)

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.AddCommand(ecscmd.EcsCmd)
	rootCmd.AddCommand(configureCmd)

	cliflags.Register(rootCmd)
	rootCmd.AddCommand(incidentscmd.Cmd(incidentscmd.DefaultClientFor(viper.GetString)))
	rootCmd.AddCommand(schemacmd.Cmd())

	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug mode")
	_ = rootCmd.PersistentFlags().MarkHidden("debug")
	rootCmd.PersistentFlags().Bool("skip-update-check", false, "Skip checking for updates")
	_ = rootCmd.PersistentFlags().MarkHidden("skip-update-check")

	config.ApplyDefaults()
	_ = viper.BindEnv("api-url", "ARMO_API_URL")
	_ = viper.BindEnv("customer-guid", "ARMO_CUSTOMER_GUID")
	_ = viper.BindEnv("access-key", "ARMO_ACCESS_KEY")
}
```

- [ ] **Step 2: Build the binary**

Run: `go build ./...`
Expected: success.

- [ ] **Step 3: Smoke-run help**

Run: `./armoctl --help` (after `go build -o armoctl .`)
Expected: `incidents` and `schema` show up in subcommand list. `armoctl incidents --help` shows `list/get/alerts/explain/resolve/unresolve/severities/fields`.

- [ ] **Step 4: Run all tests**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add root.go
git commit -m "root: wire incidents and schema clusters and shared flags"
```

---

## Task 24: End-to-end test against an httptest server

**Files:**
- Create: `cmd/incidents/incidents_e2e_test.go`

- [ ] **Step 1: Write the test**

```go
package incidents

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/spf13/cobra"
)

// TestE2E_TriageFlow exercises list → get → resolve --dry-run → resolve --yes.
func TestE2E_TriageFlow(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/runtime/incidents", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": []map[string]any{{"guid": "i1", "name": "X", "severity": "high", "status": "open", "noise": "n"}},
			"total":    map[string]any{"value": 1},
		})
	})
	mux.HandleFunc("/api/v1/runtime/incidents/i1", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"guid": "i1", "name": "X"})
	})
	mux.HandleFunc("/api/v1/runtime/incidents/i1/resolve", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	t.Setenv("ARMOCTL_AUDIT_LOG", t.TempDir()+"/audit.log")
	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})

	build := func() *cobra.Command {
		root := &cobra.Command{Use: "armoctl"}
		cliflags.Register(root)
		root.AddCommand(Cmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
		return root
	}

	// list
	{
		root := build()
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetArgs([]string{"incidents", "list"})
		if err := root.ExecuteContext(context.Background()); err != nil {
			t.Fatalf("list: %v", err)
		}
		if !strings.Contains(out.String(), "i1") || strings.Contains(out.String(), "noise") {
			t.Fatalf("list output: %s", out.String())
		}
	}
	// get
	{
		root := build()
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetArgs([]string{"incidents", "get", "i1", "--full"})
		if err := root.ExecuteContext(context.Background()); err != nil {
			t.Fatalf("get: %v", err)
		}
		if !strings.Contains(out.String(), `"guid": "i1"`) {
			t.Fatalf("get output: %s", out.String())
		}
	}
	// resolve --dry-run
	{
		root := build()
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetArgs([]string{"incidents", "resolve", "i1", "--reason", "fp", "--dry-run"})
		if err := root.ExecuteContext(context.Background()); err != nil {
			t.Fatalf("dry-run: %v", err)
		}
		if !strings.Contains(out.String(), `"dryRun"`) {
			t.Fatalf("dry-run output: %s", out.String())
		}
	}
	// resolve --yes
	{
		root := build()
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetArgs([]string{"incidents", "resolve", "i1", "--reason", "fp", "--yes"})
		if err := root.ExecuteContext(context.Background()); err != nil {
			t.Fatalf("yes: %v", err)
		}
		if !strings.Contains(out.String(), `"changed": true`) {
			t.Fatalf("yes output: %s", out.String())
		}
	}
}
```

- [ ] **Step 2: Run test — expect PASS**

Run: `go test ./cmd/incidents/... -run TestE2E -v`
Expected: PASS.

- [ ] **Step 3: Run full suite**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 4: Build the CLI**

Run: `go build -o /tmp/armoctl . && /tmp/armoctl incidents --help`
Expected: `armoctl incidents` lists every subcommand wired in this plan.

- [ ] **Step 5: Commit**

```bash
git add cmd/incidents/incidents_e2e_test.go
git commit -m "incidents: end-to-end triage flow test"
```

---

## Task 25: Extend `armoctl configure` with whoami ping

**Files:**
- Modify: `internal/config/config.go`
- Test: `internal/config/config_test.go`

- [ ] **Step 1: Write the failing test**

Append to `config_test.go`:
```go
import (
	"context"
	"net/http"
	"net/http/httptest"
)

func TestWhoami_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "K" {
			t.Errorf("missing key")
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte("{}"))
	}))
	defer srv.Close()
	if err := Whoami(context.Background(), srv.URL, "G", "K"); err != nil {
		t.Fatal(err)
	}
}

func TestWhoami_BadKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	}))
	defer srv.Close()
	if err := Whoami(context.Background(), srv.URL, "G", "K"); err == nil {
		t.Fatal("expected error")
	}
}
```

- [ ] **Step 2: Run test — expect FAIL**

Run: `go test ./internal/config/... -v`
Expected: FAIL — `Whoami` undefined.

- [ ] **Step 3: Implement**

Append to `internal/config/config.go`:
```go
// Whoami pings a lightweight read endpoint to validate that
// (apiURL, customerGUID, accessKey) form a working triple.
func Whoami(ctx context.Context, apiURL, customerGUID, accessKey string) error {
	c := apiclient.New(apiclient.Config{
		BaseURL:      apiURL,
		AccessKey:    accessKey,
		CustomerGUID: customerGUID,
	})
	var ignore map[string]any
	return c.GetJSON(ctx, "/customerState/onboarding", nil, &ignore)
}
```

Add imports `"context"` and `"github.com/armosec/armoctl/internal/apiclient"`.

In `PromptAllCredentials`, after `SaveConfig`, add:
```go
if err := Whoami(context.Background(), apiURL, guid, key); err != nil {
	fmt.Fprintf(os.Stderr, "Warning: credentials saved but whoami ping failed: %v\n", err)
}
```

- [ ] **Step 4: Run test — expect PASS**

Run: `go test ./internal/config/... -v && go test ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/config
git commit -m "config: Whoami ping in configure flow"
```

---

## Self-Review Checklist (run after writing the plan, before handoff)

**Spec coverage** — every relevant spec section has a task:

| Spec § | Coverage |
|---|---|
| §5 Architecture (apiclient/output/safety/schema/CLI) | Tasks 2–14 |
| §6 Catalog: incidents | Tasks 15–22 |
| §6 Catalog: other clusters | Out of scope (separate plans) |
| §7 Resource-verb structure | Established in Tasks 15+ (cobra layout) |
| §8 Output contract | Tasks 6–9 |
| §8.4 Token-efficient data shaping (`--fields`/`--full`/summary) | Task 9 |
| §9 Pagination | Task 16 |
| §10 Mutation safety | Tasks 10–11, 21–22 |
| §11 Errors / exit codes | Task 2 |
| §12 Auth / configure / Whoami | Tasks 1, 25 |
| §13 Schema introspection + `<resource> fields` cheatsheet | Tasks 12–13, 15 |
| §14 SKILL.md | Out of scope (separate plan) |
| §15 Plugin packaging | Out of scope (separate plan) |
| §16 Testing | Tasks 2–25 are TDD; Task 24 is e2e |
| §17 Implementation order | Tasks 2–14 = scaffolding; 15–24 = reference cluster; 25 = polish |

**Placeholder scan** — none. All code blocks are concrete.

**Type consistency** — `ClientFor`, `Cmd`, `ListCmd`/`GetCmd`/`ResolveCmd`, `safety.Args`/`safety.ExecMeta`, `output.List`/`Get`/`Mutation`/`Options`, `apiclient.ListOpts`/`PagedResult`, `clierr.Error`/`Code`/exit constants — all referenced consistently across tasks.

---

## Execution Handoff

Plan saved to `armoctl/docs/superpowers/plans/2026-04-30-armoctl-foundation-and-incidents.md`.

Two execution options:

1. **Subagent-Driven (recommended)** — fresh subagent per task, review between tasks, fast iteration.
2. **Inline Execution** — execute tasks in this session using executing-plans, batch with checkpoints.

Which approach?
