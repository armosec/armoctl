package repoposture

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

func makeServer(t *testing.T, wantPathSuffix string, wantMethod string, response any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != wantMethod {
			t.Errorf("method: got %s, want %s", r.Method, wantMethod)
		}
		if !strings.HasSuffix(r.URL.Path, wantPathSuffix) {
			t.Errorf("path: got %s, want suffix %s", r.URL.Path, wantPathSuffix)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["pageNum"] == nil || body["pageSize"] == nil {
			t.Errorf("body missing pageNum/pageSize: %v", body)
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
}

func newRoot(clientFor func(*cobra.Command) *apiclient.Client) (*cobra.Command, *bytes.Buffer) {
	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	return root, &stdout
}

func TestRepositories_PostsList(t *testing.T) {
	srv := makeServer(t, "/repositoryPosture/repositories", http.MethodPost, map[string]any{
		"response": []map[string]any{{"name": "my-repo", "owner": "acme", "provider": "github"}},
		"total":    map[string]any{"value": 1},
	})
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root, stdout := newRoot(nil)
	root.AddCommand(RepositoriesCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	root.SetArgs([]string{"repositories"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "my-repo") {
		t.Fatalf("output missing expected field: %s", stdout.String())
	}
}

func TestFiles_FilterByRepository(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/repositoryPosture/files") {
			t.Errorf("path: %s", r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["pageNum"] == nil || body["pageSize"] == nil {
			t.Errorf("body missing pageNum/pageSize: %v", body)
		}
		fl, _ := body["innerFilters"].([]any)
		if len(fl) != 1 {
			t.Errorf("innerFilters: %v", body["innerFilters"])
		}
		filters := fl[0].(map[string]any)
		if filters["repoName"] != "foo" {
			t.Errorf("innerFilters repoName: got %v, want foo", filters["repoName"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": []map[string]any{{"name": "main.yaml", "path": "/k8s/main.yaml", "type": "yaml"}},
			"total":    map[string]any{"value": 1},
		})
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	root.AddCommand(FilesCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))

	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"files", "--repository", "foo"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "main.yaml") {
		t.Fatalf("output: %s", stdout.String())
	}
}

func TestResources_PostsList(t *testing.T) {
	srv := makeServer(t, "/repositoryPosture/resources", http.MethodPost, map[string]any{
		"response": []map[string]any{{"name": "my-deployment", "kind": "Deployment", "filePath": "/k8s/deploy.yaml"}},
		"total":    map[string]any{"value": 1},
	})
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root, stdout := newRoot(nil)
	root.AddCommand(ResourcesCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	root.SetArgs([]string{"resources"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "my-deployment") {
		t.Fatalf("output missing expected field: %s", stdout.String())
	}
}

func TestFailedControls_PostsList(t *testing.T) {
	// The live endpoint requires reportGUID and kind as top-level body fields,
	// and returns a bare JSON array (not a paged envelope).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: got %s, want POST", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/repositoryPosture/failedControls") {
			t.Errorf("path: got %s, want suffix /repositoryPosture/failedControls", r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["reportGUID"] != "report-xyz" {
			t.Errorf("reportGUID: got %v, want report-xyz", body["reportGUID"])
		}
		if body["kind"] != "repo" {
			t.Errorf("kind: got %v, want repo", body["kind"])
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{"id": "C-0001", "name": "cert-rotation", "severity": "high", "framework": "security"},
		})
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root, stdout := newRoot(nil)
	root.AddCommand(FailedControlsCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	root.SetArgs([]string{"failed-controls", "--report-guid", "report-xyz"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "C-0001") {
		t.Fatalf("output missing expected field: %s", stdout.String())
	}
}
