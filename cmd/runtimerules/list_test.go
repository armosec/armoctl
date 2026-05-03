package runtimerules

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

func TestList_PostsToRuntimeRulesList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/runtime/rules/list" {
			t.Errorf("expected /api/v1/runtime/rules/list, got %s", r.URL.Path)
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
				{"guid": "r1", "name": "Rule One", "description": "First rule", "ruleType": "Custom", "createdBy": "user1", "creationTime": "2025-01-01T00:00:00Z", "policyTypes": []string{"ADR"}},
				{"guid": "r2", "name": "Rule Two", "description": "Second rule", "ruleType": "Managed", "createdBy": "system", "creationTime": "2025-01-02T00:00:00Z", "policyTypes": []string{"CDR"}},
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
	if !strings.Contains(out, `"items"`) || !strings.Contains(out, "r1") || !strings.Contains(out, "r2") {
		t.Fatalf("unexpected list output: %s", out)
	}
}

func TestList_NameFilterAddedToInnerFilters(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		filters, ok := body["innerFilters"].([]any)
		if !ok || len(filters) != 1 {
			t.Errorf("expected innerFilters with 1 item: %v", body)
		}
		filterMap, _ := filters[0].(map[string]any)
		if filterMap["name"] != "test-rule" {
			t.Errorf("expected name filter 'test-rule': %v", filterMap)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": []map[string]any{},
			"total":    map[string]any{"value": 0},
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
	root.SetArgs([]string{"list", "--name", "test-rule"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
}
