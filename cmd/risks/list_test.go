package risks

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

func TestList_PostsList(t *testing.T) {
	srv := makeServer(t, "/securityrisks/list", http.MethodPost, map[string]any{
		"response": []map[string]any{{"name": "CVE-2024-123", "severity": "critical", "id": "risk-1"}},
		"total":    map[string]any{"value": 1},
	})
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root, stdout := newRoot(nil)
	root.AddCommand(ListCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	root.SetArgs([]string{"list"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "CVE-2024-123") {
		t.Fatalf("output missing expected field: %s", stdout.String())
	}
}

func TestList_FiltersBySeverity(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/securityrisks/list") {
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
			"response": []map[string]any{{"name": "risk-1", "severity": "critical"}},
			"total":    map[string]any{"value": 1},
		})
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	root.AddCommand(ListCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))

	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"list", "--severity", "critical"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "risk-1") {
		t.Fatalf("output: %s", stdout.String())
	}
}

func TestResources_PostsList(t *testing.T) {
	// The live endpoint requires securityRiskID in innerFilters.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: got %s, want POST", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/securityrisks/resources") {
			t.Errorf("path: got %s, want suffix /securityrisks/resources", r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["pageNum"] == nil || body["pageSize"] == nil {
			t.Errorf("body missing pageNum/pageSize: %v", body)
		}
		fl, _ := body["innerFilters"].([]any)
		if len(fl) == 0 {
			t.Errorf("innerFilters missing: %v", body)
		} else {
			f := fl[0].(map[string]any)
			if f["securityRiskID"] != "risk-abc" {
				t.Errorf("securityRiskID: got %v, want risk-abc", f["securityRiskID"])
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": []map[string]any{{"name": "my-app", "namespace": "default", "kind": "Deployment"}},
			"total":    map[string]any{"value": 1},
		})
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root, stdout := newRoot(nil)
	root.AddCommand(ResourcesCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	root.SetArgs([]string{"resources", "--risk-id", "risk-abc"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "my-app") {
		t.Fatalf("output missing expected field: %s", stdout.String())
	}
}

func TestSeverities_PostsList(t *testing.T) {
	// The live endpoint returns a bare object (no response/total envelope).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: got %s, want POST", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/securityrisks/severities") {
			t.Errorf("path: got %s, want suffix /securityrisks/severities", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": map[string]any{
				"severityResourceCounter": map[string]any{"Critical": 268, "High": 457},
				"totalResources":          725,
			},
			"total": map[string]any{"value": 4, "relation": "eq"},
		})
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root, stdout := newRoot(nil)
	root.AddCommand(SeveritiesCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	root.SetArgs([]string{"severities"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Critical") {
		t.Fatalf("output missing expected field: %s", stdout.String())
	}
}
