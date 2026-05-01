package posture

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
		if !strings.HasSuffix(r.URL.Path, "/postureExceptionPolicy") {
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
		"--control", "C-0001",
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

	if body["policyType"] != "postureExceptionPolicy" {
		t.Errorf("policyType: got %v", body["policyType"])
	}

	actions, _ := body["actions"].([]any)
	if len(actions) != 1 || actions[0] != "alertOnly" {
		t.Errorf("actions: got %v", body["actions"])
	}

	policies, _ := body["posturePolicies"].([]any)
	if len(policies) != 1 {
		t.Fatalf("posturePolicies: got %v", body["posturePolicies"])
	}
	p0, _ := policies[0].(map[string]any)
	if p0["controlID"] != "C-0001" {
		t.Errorf("policy controlID: got %v", p0["controlID"])
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
		if !strings.HasSuffix(r.URL.Path, "/postureExceptionPolicy") {
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
		"--control", "C-0001",
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
	if capturedBody["policyType"] != "postureExceptionPolicy" {
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

func TestExceptionsCreate_NoControlFails(t *testing.T) {
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
		t.Fatal("expected error when no --control provided, got nil")
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
		"--control", "C-0001",
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
	if !strings.HasSuffix(capturedPath, "/api/v1/postureExceptionPolicy") {
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
