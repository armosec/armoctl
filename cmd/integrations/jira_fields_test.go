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
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/spf13/cobra"
)

func TestJiraFields_RequiresProjectAndIssueType(t *testing.T) {
	c := apiclient.New(apiclient.Config{BaseURL: "http://localhost", AccessKey: "K", CustomerGUID: "G"})
	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	var stdout bytes.Buffer
	root.SetOut(&stdout)

	integ := &cobra.Command{Use: "integrations"}
	jira := &cobra.Command{Use: "jira"}
	jira.AddCommand(JiraFieldsCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	integ.AddCommand(jira)
	root.AddCommand(integ)

	root.SetArgs([]string{"integrations", "jira", "fields"})
	err := root.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error when --project and --issue-type not provided")
	}
	var ce *clierr.Error
	if !isCliErr(err, &ce) {
		t.Fatalf("expected *clierr.Error, got %T: %v", err, err)
	}
	if ce.Code != clierr.CodeBadInput {
		t.Errorf("code: got %v, want CodeBadInput", ce.Code)
	}
}

func TestJiraFields_PostsBody(t *testing.T) {
	var capturedBody map[string]any
	var hits int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		_ = json.NewDecoder(r.Body).Decode(&capturedBody)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": []map[string]any{
				{"name": "summary", "type": "string", "required": true},
				{"name": "description", "type": "string", "required": false},
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
	jira.AddCommand(JiraFieldsCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	integ.AddCommand(jira)
	root.AddCommand(integ)

	root.SetArgs([]string{"integrations", "jira", "fields", "--project", "SEC", "--issue-type", "BUG123"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}

	if atomic.LoadInt32(&hits) != 1 {
		t.Errorf("expected 1 server hit, got %d", hits)
	}
	if capturedBody["projectKey"] != "SEC" {
		t.Errorf("projectKey: got %v, want SEC", capturedBody["projectKey"])
	}
	if capturedBody["issueTypeID"] != "BUG123" {
		t.Errorf("issueTypeID: got %v, want BUG123", capturedBody["issueTypeID"])
	}

	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("output not JSON: %v", err)
	}
}

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
