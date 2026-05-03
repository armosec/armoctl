package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/spf13/cobra"
)

func TestJiraIssueTypes_FilterByProject(t *testing.T) {
	var capturedBody map[string]any
	var hits int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		_ = json.NewDecoder(r.Body).Decode(&capturedBody)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": []map[string]any{
				{"id": "1", "name": "Bug", "description": "A bug in the software", "subtask": false},
				{"id": "2", "name": "Task", "description": "A task", "subtask": false},
			},
			"total": map[string]int{"value": 2},
		})
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	var stdout bytes.Buffer
	root.SetOut(&stdout)

	integ := &cobra.Command{Use: "integrations"}
	jira := &cobra.Command{Use: "jira"}
	jira.AddCommand(JiraIssueTypesCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	integ.AddCommand(jira)
	root.AddCommand(integ)

	root.SetArgs([]string{"integrations", "jira", "issue-types", "--project", "SEC"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}

	if atomic.LoadInt32(&hits) != 1 {
		t.Errorf("expected 1 server hit, got %d", hits)
	}
	if capturedBody["projectKey"] != "SEC" {
		t.Errorf("projectKey in body: got %v, want SEC", capturedBody["projectKey"])
	}

	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("output not JSON: %v", err)
	}
	items, _ := result["items"].([]any)
	if len(items) != 2 {
		t.Errorf("expected 2 items in output, got %d", len(items))
	}
}

func TestJiraIssueTypes_PostsWithoutProject(t *testing.T) {
	var hits int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": []map[string]any{},
			"total":    map[string]int{"value": 0},
		})
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	var stdout bytes.Buffer
	root.SetOut(&stdout)

	integ := &cobra.Command{Use: "integrations"}
	jira := &cobra.Command{Use: "jira"}
	jira.AddCommand(JiraIssueTypesCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	integ.AddCommand(jira)
	root.AddCommand(integ)

	root.SetArgs([]string{"integrations", "jira", "issue-types"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}

	if atomic.LoadInt32(&hits) != 1 {
		t.Errorf("expected 1 server hit, got %d", hits)
	}

	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("output not JSON: %v", err)
	}
}
