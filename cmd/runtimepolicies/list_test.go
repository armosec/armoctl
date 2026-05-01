package runtimepolicies

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

func TestList_PostsToRuntimePolicies(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/runtime/policies" {
			t.Errorf("expected /api/v1/runtime/policies, got %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if _, ok := body["pageNum"]; !ok {
			t.Errorf("body missing pageNum: %v", body)
		}
		if _, ok := body["pageSize"]; !ok {
			t.Errorf("body missing pageSize: %v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": []map[string]any{
				{"guid": "p1", "name": "Policy One", "description": "First policy", "enabled": true, "scope": map[string]any{}, "creationTime": "2025-01-01T00:00:00Z"},
				{"guid": "p2", "name": "Policy Two", "description": "Second policy", "enabled": false, "scope": map[string]any{}, "creationTime": "2025-01-02T00:00:00Z"},
			},
			"total": map[string]any{"value": 2},
		})
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})

	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	root.AddCommand(Cmd(func(cmd *cobra.Command) *apiclient.Client { return c }))

	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stdout)
	root.SetArgs([]string{"runtime-policies", "list"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	out := stdout.String()
	if !strings.Contains(out, `"items"`) || !strings.Contains(out, "p1") || !strings.Contains(out, "p2") {
		t.Fatalf("unexpected list output: %s", out)
	}
}
