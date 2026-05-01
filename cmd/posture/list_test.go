package posture

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

func TestFrameworks_PostsList(t *testing.T) {
	srv := makeServer(t, "/posture/frameworks", http.MethodPost, map[string]any{
		"response": []map[string]any{{"name": "NSA", "complianceScore": 0.85}},
		"total":    map[string]any{"value": 1},
	})
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root, stdout := newRoot(nil)
	root.AddCommand(FrameworksCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	root.SetArgs([]string{"frameworks"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "NSA") {
		t.Fatalf("output missing expected field: %s", stdout.String())
	}
}

func TestControls_FiltersByFramework(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/posture/controls") {
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
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": []map[string]any{{"name": "C-0001", "framework": "NSA", "id": "C-0001"}},
			"total":    map[string]any{"value": 1},
		})
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	root.AddCommand(ControlsCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))

	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"controls", "--framework", "NSA"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "C-0001") {
		t.Fatalf("output: %s", stdout.String())
	}
}

func TestResources_PostsList(t *testing.T) {
	srv := makeServer(t, "/posture/resources", http.MethodPost, map[string]any{
		"response": []map[string]any{{"name": "my-app", "namespace": "default", "kind": "Deployment"}},
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
	if !strings.Contains(stdout.String(), "my-app") {
		t.Fatalf("output missing expected field: %s", stdout.String())
	}
}
