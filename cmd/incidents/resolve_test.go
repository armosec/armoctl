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
	root.SetArgs([]string{"resolve", "i1", "--reason", "fp", "--dry-run"})
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
}

func TestResolve_YesPostsAndReportsChanged(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method: %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/runtime/incidents/i1/resolve") {
			t.Errorf("path: %s", r.URL.Path)
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
	root.SetArgs([]string{"resolve", "i1", "--reason", "fp", "--yes"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), `"changed": true`) {
		t.Fatalf("changed not set: %s", stdout.String())
	}
}
