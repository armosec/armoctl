package incidents

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/spf13/cobra"
)

func TestParseFilters(t *testing.T) {
	got, err := parseFilters([]string{"severity=Low", "clusterName=prod"})
	if err != nil {
		t.Fatal(err)
	}
	want := []map[string]string{{"severity": "Low", "clusterName": "prod"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseFilters = %v, want %v", got, want)
	}
	if _, err := parseFilters([]string{"bad-no-eq"}); err == nil {
		t.Fatal("expected error for filter without '='")
	}
	if got, _ := parseFilters(nil); got != nil {
		t.Fatalf("parseFilters(nil) = %v, want nil", got)
	}
}

// newRoot wires a fresh root command with mutation flags and set-status.
func newRoot(c *apiclient.Client) (*cobra.Command, *bytes.Buffer) {
	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	root.AddCommand(SetStatusCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	return root, &stdout
}

func TestSetStatus_DryRunBody(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(200)
		_, _ = w.Write([]byte("{}"))
	}))
	defer srv.Close()
	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})

	root, stdout := newRoot(c)
	root.SetArgs([]string{"set-status", "i1", "i2", "--status", "Dismissed", "--dry-run"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&hits) != 0 {
		t.Fatal("server called during dry-run")
	}
	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("output not JSON: %v: %q", err, stdout.String())
	}
	req, _ := got["request"].(map[string]any)
	body, _ := req["body"].(map[string]any)
	if body["status"] != "Dismissed" {
		t.Fatalf("status: %v", body["status"])
	}
	guids, _ := body["incidentsGuids"].([]any)
	if len(guids) != 2 || guids[0] != "i1" || guids[1] != "i2" {
		t.Fatalf("incidentsGuids: %v", body["incidentsGuids"])
	}
}

func TestSetStatus_FilterAndSearch(t *testing.T) {
	c := apiclient.New(apiclient.Config{BaseURL: "http://unused", AccessKey: "K", CustomerGUID: "G"})
	root, stdout := newRoot(c)
	root.SetArgs([]string{
		"set-status", "--status", "Dismissed",
		"--filter", "severity=Low", "--filter", "clusterName=prod",
		"--search", "nginx", "--dry-run",
	})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	_ = json.Unmarshal(stdout.Bytes(), &got)
	req, _ := got["request"].(map[string]any)
	body, _ := req["body"].(map[string]any)
	filters, _ := body["innerFilters"].([]any)
	if len(filters) != 1 {
		t.Fatalf("innerFilters: %v", body["innerFilters"])
	}
	f0, _ := filters[0].(map[string]any)
	if f0["severity"] != "Low" || f0["clusterName"] != "prod" {
		t.Fatalf("filter map: %v", f0)
	}
	q, _ := req["query"].(map[string]any)
	if q["searchText"] != "nginx" {
		t.Fatalf("query searchText: %v", req["query"])
	}
}

func TestSetStatus_Stdin(t *testing.T) {
	c := apiclient.New(apiclient.Config{BaseURL: "http://unused", AccessKey: "K", CustomerGUID: "G"})
	root, stdout := newRoot(c)
	root.SetIn(strings.NewReader("i1 i2\ni3\n"))
	root.SetArgs([]string{"set-status", "--status", "Resolved", "--stdin", "--dry-run"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	_ = json.Unmarshal(stdout.Bytes(), &got)
	req, _ := got["request"].(map[string]any)
	body, _ := req["body"].(map[string]any)
	guids, _ := body["incidentsGuids"].([]any)
	if len(guids) != 3 {
		t.Fatalf("expected 3 guids from stdin, got %v", body["incidentsGuids"])
	}
}

func TestSetStatus_InvalidStatus(t *testing.T) {
	c := apiclient.New(apiclient.Config{BaseURL: "http://unused", AccessKey: "K", CustomerGUID: "G"})
	root, _ := newRoot(c)
	root.SetArgs([]string{"set-status", "i1", "--status", "Bogus", "--dry-run"})
	if err := root.ExecuteContext(context.Background()); err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestSetStatus_NoSelection(t *testing.T) {
	c := apiclient.New(apiclient.Config{BaseURL: "http://unused", AccessKey: "K", CustomerGUID: "G"})
	root, _ := newRoot(c)
	root.SetArgs([]string{"set-status", "--status", "Dismissed", "--dry-run"})
	if err := root.ExecuteContext(context.Background()); err == nil {
		t.Fatal("expected error when no incidents selected")
	}
}

func TestSetStatus_YesPosts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/runtime/incidents/changeStatus" {
			t.Errorf("path: %s", r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["status"] != "Dismissed" {
			t.Errorf("body.status: %v", body["status"])
		}
		w.Header().Set("x-request-id", "req-1")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"changed":true}`))
	}))
	defer srv.Close()
	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})

	t.Setenv("ARMOCTL_AUDIT_LOG", t.TempDir()+"/audit.log")

	root, stdout := newRoot(c)
	root.SetArgs([]string{"set-status", "i1", "--status", "Dismissed", "--yes"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), `"changed": true`) {
		t.Fatalf("changed not set: %s", stdout.String())
	}
}
