package incidents

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

func TestResolve_DryRunDoesNotCallServer(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(200)
		_, _ = w.Write([]byte("{}"))
	}))
	defer srv.Close()
	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})

	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	root.AddCommand(ResolveCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))

	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"resolve", "i1", "--dry-run"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&hits) != 0 {
		t.Fatalf("server was called during dry-run")
	}
	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("output not JSON: %v: %q", err, stdout.String())
	}
	if got["dryRun"] != true {
		t.Fatalf("dryRun: %v", got)
	}
	req, _ := got["request"].(map[string]any)
	if req == nil || req["url"] != "/runtime/incidents/changeStatus" {
		t.Fatalf("preview url: %v", got)
	}
	body, _ := req["body"].(map[string]any)
	guids, _ := body["incidentsGuids"].([]any)
	if len(guids) != 1 || guids[0] != "i1" {
		t.Fatalf("preview body incidentsGuids: %v", body)
	}
	if body["status"] != "Resolved" {
		t.Fatalf("preview body status: %v", body)
	}
}

func TestResolve_YesPostsAndReportsChanged(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method: %s", r.Method)
		}
		if r.URL.Path != "/api/v1/runtime/incidents/changeStatus" {
			t.Errorf("path: %s", r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["status"] != "Resolved" {
			t.Errorf("body.status: %v", body["status"])
		}
		guids, _ := body["incidentsGuids"].([]any)
		if len(guids) != 1 || guids[0] != "i1" {
			t.Errorf("body.incidentsGuids: %v", body["incidentsGuids"])
		}
		w.Header().Set("x-request-id", "req-z")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"resolved":true}`))
	}))
	defer srv.Close()
	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})

	t.Setenv("ARMOCTL_AUDIT_LOG", t.TempDir()+"/audit.log")

	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	root.AddCommand(ResolveCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))

	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"resolve", "i1", "--yes"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), `"changed": true`) {
		t.Fatalf("changed not set: %s", stdout.String())
	}
}
