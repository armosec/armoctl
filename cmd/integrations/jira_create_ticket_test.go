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

func TestCreateTicket_DryRun(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	var stdout bytes.Buffer
	root.SetOut(&stdout)

	integ := &cobra.Command{Use: "integrations"}
	jira := &cobra.Command{Use: "jira"}
	jira.AddCommand(JiraCreateTicketCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	integ.AddCommand(jira)
	root.AddCommand(integ)

	root.SetArgs([]string{
		"integrations", "jira", "create-ticket",
		"--project", "SEC",
		"--issue-type", "Bug",
		"--summary", "Test bug",
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
		t.Fatalf("output not JSON: %v", err)
	}

	dryRun, _ := result["dryRun"].(bool)
	if !dryRun {
		t.Errorf("expected dryRun=true in output")
	}
}

func TestCreateTicket_Yes(t *testing.T) {
	var capturedBody map[string]any
	var hits int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		_ = json.NewDecoder(r.Body).Decode(&capturedBody)
		_ = json.NewEncoder(w).Encode(map[string]any{"key": "SEC-123", "id": "123"})
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	var stdout bytes.Buffer
	root.SetOut(&stdout)

	integ := &cobra.Command{Use: "integrations"}
	jira := &cobra.Command{Use: "jira"}
	jira.AddCommand(JiraCreateTicketCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	integ.AddCommand(jira)
	root.AddCommand(integ)

	root.SetArgs([]string{
		"integrations", "jira", "create-ticket",
		"--project", "SEC",
		"--issue-type", "Bug",
		"--summary", "Test bug",
		"--description", "A test bug",
		"--field", "customField=value",
		"--yes",
	})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}

	if atomic.LoadInt32(&hits) != 1 {
		t.Errorf("expected 1 server hit, got %d", hits)
	}

	if capturedBody["projectKey"] != "SEC" {
		t.Errorf("projectKey: got %v, want SEC", capturedBody["projectKey"])
	}
	if capturedBody["issueTypeName"] != "Bug" {
		t.Errorf("issueTypeName: got %v, want Bug", capturedBody["issueTypeName"])
	}

	fields, _ := capturedBody["fields"].(map[string]any)
	if fields["summary"] != "Test bug" {
		t.Errorf("summary: got %v, want 'Test bug'", fields["summary"])
	}
	if fields["description"] != "A test bug" {
		t.Errorf("description: got %v, want 'A test bug'", fields["description"])
	}
	if fields["customField"] != "value" {
		t.Errorf("customField: got %v, want 'value'", fields["customField"])
	}

	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("output not JSON: %v", err)
	}
	changed, _ := result["changed"].(bool)
	if !changed {
		t.Errorf("expected changed=true in output")
	}
}

func TestCreateTicket_RequiresFields(t *testing.T) {
	c := apiclient.New(apiclient.Config{BaseURL: "http://localhost", AccessKey: "K", CustomerGUID: "G"})
	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	var stdout bytes.Buffer
	root.SetOut(&stdout)

	integ := &cobra.Command{Use: "integrations"}
	jira := &cobra.Command{Use: "jira"}
	jira.AddCommand(JiraCreateTicketCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	integ.AddCommand(jira)
	root.AddCommand(integ)

	root.SetArgs([]string{
		"integrations", "jira", "create-ticket",
		"--project", "SEC",
		"--yes",
	})
	err := root.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error when --issue-type or --summary not provided")
	}
	var ce *clierr.Error
	if !isCliErr(err, &ce) {
		t.Fatalf("expected *clierr.Error, got %T: %v", err, err)
	}
	if ce.Code != clierr.CodeBadInput {
		t.Errorf("code: got %v, want CodeBadInput", ce.Code)
	}
}
