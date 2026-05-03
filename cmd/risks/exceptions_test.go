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
