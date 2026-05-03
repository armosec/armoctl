package risks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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
