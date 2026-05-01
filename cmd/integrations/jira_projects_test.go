package integrations

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

func TestJiraProjects_Posts(t *testing.T) {
	var capturedMethod string
	var capturedPath string
	var capturedBody map[string]any
	var hits int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&capturedBody)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": []map[string]any{
				{"key": "SEC", "name": "Security", "id": "123", "projectTypeKey": "software", "lead": "john"},
			},
			"total": map[string]int{"value": 1},
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
	jira.AddCommand(JiraProjectsCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	integ.AddCommand(jira)
	root.AddCommand(integ)

	root.SetArgs([]string{"integrations", "jira", "projects"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}

	if atomic.LoadInt32(&hits) != 1 {
		t.Errorf("expected 1 server hit, got %d", hits)
	}
	if capturedMethod != http.MethodPost {
		t.Errorf("method: got %s, want POST", capturedMethod)
	}
	if !strings.HasSuffix(capturedPath, "/integrations/jira/projectsV2/search") {
		t.Errorf("path: got %s", capturedPath)
	}

	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("output not JSON: %v", err)
	}
	items, _ := result["items"].([]any)
	if len(items) != 1 {
		t.Errorf("expected 1 item in output, got %d", len(items))
	}
}
