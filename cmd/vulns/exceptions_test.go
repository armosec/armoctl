package vulns

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

func newExcRoot(clientFor func(*cobra.Command) *apiclient.Client) (*cobra.Command, *bytes.Buffer) {
	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	return root, &stdout
}

func TestExceptionsList_FlattensArray(t *testing.T) {
	policies := []map[string]any{
		{"guid": "g1", "name": "pol-1", "creationTime": "2024-01-01T00:00:00Z"},
		{"guid": "g2", "name": "pol-2", "creationTime": "2024-06-15T00:00:00Z"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method: got %s, want GET", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/vulnerabilityExceptionPolicy") {
			t.Errorf("path: got %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(policies)
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root, stdout := newExcRoot(nil)
	exc := &cobra.Command{Use: "exceptions"}
	exc.AddCommand(ExceptionsListCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	root.AddCommand(exc)
	root.SetArgs([]string{"exceptions", "list"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}

	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("output not JSON: %v — %s", err, stdout.String())
	}
	total, _ := result["total"].(float64)
	if int(total) != 2 {
		t.Errorf("total: got %v, want 2", total)
	}
	items, _ := result["items"].([]any)
	if len(items) != 2 {
		t.Errorf("items count: got %d, want 2", len(items))
	}
}

func TestExceptionsCreate_DryRun(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root, stdout := newExcRoot(nil)
	exc := &cobra.Command{Use: "exceptions"}
	exc.AddCommand(ExceptionsCreateCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	root.AddCommand(exc)
	root.SetArgs([]string{
		"exceptions", "create",
		"--name", "my-exception",
		"--cve", "CVE-2024-1234",
		"--cluster", "X",
		"--dry-run",
	})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}

	if atomic.LoadInt32(&hits) != 0 {
		t.Errorf("server was contacted during dry-run (%d hits)", hits)
	}

	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("output not JSON: %v — %s", err, stdout.String())
	}

	dryRun, _ := result["dryRun"].(bool)
	if !dryRun {
		t.Errorf("expected dryRun=true in output")
	}

	req, _ := result["request"].(map[string]any)
	if req == nil {
		t.Fatalf("request field missing in output: %v", result)
	}
	body, _ := req["body"].(map[string]any)
	if body == nil {
		t.Fatalf("body missing in request: %v", req)
	}

	if body["policyType"] != "vulnerabilityExceptionPolicy" {
		t.Errorf("policyType: got %v", body["policyType"])
	}

	actions, _ := body["actions"].([]any)
	if len(actions) != 1 || actions[0] != "ignore" {
		t.Errorf("actions: got %v", body["actions"])
	}

	vulns, _ := body["vulnerabilities"].([]any)
	if len(vulns) != 1 {
		t.Fatalf("vulnerabilities: got %v", body["vulnerabilities"])
	}
	v0, _ := vulns[0].(map[string]any)
	if v0["name"] != "CVE-2024-1234" {
		t.Errorf("vulnerability name: got %v", v0["name"])
	}

	designators, _ := body["designators"].([]any)
	if len(designators) != 1 {
		t.Fatalf("designators: got %v", body["designators"])
	}
	d0, _ := designators[0].(map[string]any)
	attrs, _ := d0["attributes"].(map[string]any)
	if attrs["cluster"] != "X" {
		t.Errorf("designator cluster: got %v", attrs["cluster"])
	}
}

func TestExceptionsCreate_Yes(t *testing.T) {
	var capturedBody map[string]any
	var hits int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		if r.Method != http.MethodPost {
			t.Errorf("method: got %s, want POST", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/vulnerabilityExceptionPolicy") {
			t.Errorf("path: got %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&capturedBody)
		_ = json.NewEncoder(w).Encode(map[string]any{"guid": "new-guid", "name": "my-exception"})
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root, stdout := newExcRoot(nil)
	exc := &cobra.Command{Use: "exceptions"}
	exc.AddCommand(ExceptionsCreateCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	root.AddCommand(exc)
	root.SetArgs([]string{
		"exceptions", "create",
		"--name", "my-exception",
		"--cve", "CVE-2024-1234",
		"--cluster", "prod",
		"--yes",
	})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}

	if atomic.LoadInt32(&hits) != 1 {
		t.Errorf("expected 1 server hit, got %d", hits)
	}

	// Verify request body fields
	if capturedBody["policyType"] != "vulnerabilityExceptionPolicy" {
		t.Errorf("policyType: got %v", capturedBody["policyType"])
	}

	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("output not JSON: %v — %s", err, stdout.String())
	}
	changed, _ := result["changed"].(bool)
	if !changed {
		t.Errorf("expected changed=true in output, got: %v", result)
	}
}

func TestExceptionsCreate_NoCVEFails(t *testing.T) {
	c := apiclient.New(apiclient.Config{BaseURL: "http://localhost", AccessKey: "K", CustomerGUID: "G"})
	root, _ := newExcRoot(nil)
	exc := &cobra.Command{Use: "exceptions"}
	exc.AddCommand(ExceptionsCreateCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	root.AddCommand(exc)
	root.SetArgs([]string{
		"exceptions", "create",
		"--name", "my-exception",
		"--cluster", "prod",
		"--yes",
	})
	err := root.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error when no --cve provided, got nil")
	}
	var ce *clierr.Error
	if !isCliErr(err, &ce) {
		t.Fatalf("expected *clierr.Error, got %T: %v", err, err)
	}
	if ce.Code != clierr.CodeBadInput {
		t.Errorf("code: got %v, want %v", ce.Code, clierr.CodeBadInput)
	}
}

func TestExceptionsCreate_NoDesignatorFails(t *testing.T) {
	c := apiclient.New(apiclient.Config{BaseURL: "http://localhost", AccessKey: "K", CustomerGUID: "G"})
	root, _ := newExcRoot(nil)
	exc := &cobra.Command{Use: "exceptions"}
	exc.AddCommand(ExceptionsCreateCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	root.AddCommand(exc)
	root.SetArgs([]string{
		"exceptions", "create",
		"--name", "my-exception",
		"--cve", "CVE-2024-1234",
		"--yes",
	})
	err := root.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error when no designator flags provided, got nil")
	}
	var ce *clierr.Error
	if !isCliErr(err, &ce) {
		t.Fatalf("expected *clierr.Error, got %T: %v", err, err)
	}
	if ce.Code != clierr.CodeBadInput {
		t.Errorf("code: got %v, want %v", ce.Code, clierr.CodeBadInput)
	}
}

func TestExceptionsDelete_Yes(t *testing.T) {
	var capturedMethod string
	var capturedPath string
	var capturedQuery string
	var hits int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		capturedQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`["deleted"]`))
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root, stdout := newExcRoot(nil)
	exc := &cobra.Command{Use: "exceptions"}
	exc.AddCommand(ExceptionsDeleteCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	root.AddCommand(exc)
	root.SetArgs([]string{
		"exceptions", "delete", "my-policy",
		"--yes",
	})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}

	if atomic.LoadInt32(&hits) != 1 {
		t.Errorf("expected 1 server hit, got %d", hits)
	}
	if capturedMethod != http.MethodDelete {
		t.Errorf("method: got %s, want DELETE", capturedMethod)
	}
	if !strings.HasSuffix(capturedPath, "/api/v1/vulnerabilityExceptionPolicy") {
		t.Errorf("path: got %s", capturedPath)
	}
	if !strings.Contains(capturedQuery, "policyName=my-policy") {
		t.Errorf("query: got %s, want policyName=my-policy", capturedQuery)
	}

	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("output not JSON: %v — %s", err, stdout.String())
	}
	changed, _ := result["changed"].(bool)
	if !changed {
		t.Errorf("expected changed=true in output, got: %v", result)
	}
}

func TestExceptionsUpdate_DryRun(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root, stdout := newExcRoot(nil)
	exc := &cobra.Command{Use: "exceptions"}
	exc.AddCommand(ExceptionsUpdateCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	root.AddCommand(exc)
	root.SetArgs([]string{
		"exceptions", "update",
		"--guid", "abc-123",
		"--name", "my-exception",
		"--cve", "CVE-2024-5678",
		"--cluster", "prod",
		"--dry-run",
	})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}

	if atomic.LoadInt32(&hits) != 0 {
		t.Errorf("server was contacted during dry-run (%d hits)", hits)
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
	var capturedBody map[string]any
	var hits int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		if r.Method != http.MethodPut {
			t.Errorf("method: got %s, want PUT", r.Method)
		}
		_ = json.NewDecoder(r.Body).Decode(&capturedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root, _ := newExcRoot(nil)
	exc := &cobra.Command{Use: "exceptions"}
	exc.AddCommand(ExceptionsUpdateCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	root.AddCommand(exc)
	root.SetArgs([]string{
		"exceptions", "update",
		"--guid", "abc-123",
		// intentionally omitting --name
		"--cve", "CVE-2024-5678",
		"--cluster", "prod",
		"--yes",
	})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}

	if atomic.LoadInt32(&hits) != 1 {
		t.Errorf("expected 1 server hit, got %d", hits)
	}

	if _, ok := capturedBody["name"]; ok {
		t.Errorf("expected 'name' key to be absent from body when --name not passed, got body: %v", capturedBody)
	}
	if capturedBody["guid"] != "abc-123" {
		t.Errorf("body.guid: got %v, want abc-123", capturedBody["guid"])
	}
}

// isCliErr tries to assign err into *clierr.Error via errors.As semantics manually.
func isCliErr(err error, target **clierr.Error) bool {
	if err == nil {
		return false
	}
	if ce, ok := err.(*clierr.Error); ok {
		*target = ce
		return true
	}
	return false
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

			root, _ := newExcRoot(nil)
			exc := &cobra.Command{Use: "exceptions"}
			exc.AddCommand(ExceptionsUpdateCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
			root.AddCommand(exc)
			root.SetArgs([]string{
				"exceptions", "update",
				"--guid", "abc",
				"--cve", "CVE-2024-1",
				"--cluster", "prod",
				"--yes",
			})
			err := root.ExecuteContext(context.Background())
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			var ce *clierr.Error
			if !errors.As(err, &ce) {
				t.Fatalf("error not *clierr.Error: %v", err)
			}
			if ce.Code != tc.want {
				t.Fatalf("code: got %v, want %v", ce.Code, tc.want)
			}
			// 5xx triggers apiclient retry which closes the body; the
			// final returned response has no body to extract from.
			// Skip message assertion in that case.
			if tc.status < 500 && ce.Msg != "upstream said no" {
				t.Errorf("msg: got %q, want extracted JSON message", ce.Msg)
			}
		})
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

			root, _ := newExcRoot(nil)
			exc := &cobra.Command{Use: "exceptions"}
			exc.AddCommand(ExceptionsDeleteCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
			root.AddCommand(exc)
			root.SetArgs([]string{"exceptions", "delete", "p", "--yes"})
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
			if tc.status < 500 && ce.Msg != "nope" {
				t.Errorf("msg: got %q, want extracted JSON error", ce.Msg)
			}
		})
	}
}
