package vulns

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

func TestWorkloads_ListsPaged(t *testing.T) {
	srv := makeServer(t, "/vulnerability_v2/workload/list", http.MethodPost, map[string]any{
		"response": []map[string]any{{"name": "my-deploy", "namespace": "default", "kind": "Deployment"}},
		"total":    map[string]any{"value": 1},
	})
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root, stdout := newRoot(nil)
	root.AddCommand(WorkloadsCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	root.SetArgs([]string{"workloads"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "my-deploy") {
		t.Fatalf("output missing expected field: %s", stdout.String())
	}
}

func TestImages_ListsPaged(t *testing.T) {
	srv := makeServer(t, "/vulnerability_v2/image/list", http.MethodPost, map[string]any{
		"response": []map[string]any{{"repository": "nginx", "tag": "latest", "registry": "docker.io"}},
		"total":    map[string]any{"value": 1},
	})
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root, stdout := newRoot(nil)
	root.AddCommand(ImagesCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	root.SetArgs([]string{"images"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "nginx") {
		t.Fatalf("output missing expected field: %s", stdout.String())
	}
}

func TestComponents_ListsPaged(t *testing.T) {
	srv := makeServer(t, "/vulnerability_v2/component/list", http.MethodPost, map[string]any{
		"response": []map[string]any{{"name": "openssl", "version": "3.0.1", "packageType": "deb"}},
		"total":    map[string]any{"value": 1},
	})
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root, stdout := newRoot(nil)
	root.AddCommand(ComponentsCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	root.SetArgs([]string{"components"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "openssl") {
		t.Fatalf("output missing expected field: %s", stdout.String())
	}
}

func TestCVEs_FiltersBySeverity(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/vulnerability_v2/vulnerability/list") {
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
			"response": []map[string]any{{"name": "GHSA-xxx", "severity": "Critical"}},
			"total":    map[string]any{"value": 1},
		})
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	root.AddCommand(CVEsCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))

	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"cves", "--severity", "Critical"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "GHSA-xxx") {
		t.Fatalf("output: %s", stdout.String())
	}
}

func TestHosts_ListsPaged(t *testing.T) {
	srv := makeServer(t, "/vulnerability_v2/host/list", http.MethodPost, map[string]any{
		"response": []map[string]any{{"hostName": "node-1", "hostType": "kubernetes", "region": "us-east-1"}},
		"total":    map[string]any{"value": 1},
	})
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root, stdout := newRoot(nil)
	root.AddCommand(HostsCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	root.SetArgs([]string{"hosts"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "node-1") {
		t.Fatalf("output missing expected field: %s", stdout.String())
	}
}
