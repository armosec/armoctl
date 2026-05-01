package cloudaccounts

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

func TestECSDisconnect_DryRun(t *testing.T) {
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
	root.AddCommand(ECSDisconnectCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	root.SetArgs([]string{
		"disconnect", "arn:aws:ecs:us-east-1:123456789:cluster/prod",
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
		t.Fatalf("output not JSON: %v — %s", err, stdout.String())
	}

	dryRun, _ := result["dryRun"].(bool)
	if !dryRun {
		t.Errorf("expected dryRun=true in output")
	}

	req, _ := result["request"].(map[string]any)
	if req == nil {
		t.Fatalf("request field missing in output: %v", result)
	}
}

func TestECSDisconnect_Yes(t *testing.T) {
	var capturedMethod string
	var capturedPath string
	var capturedQuery string
	var hits int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		capturedQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`["disconnected"]`))
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.AddCommand(ECSDisconnectCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	root.SetArgs([]string{
		"disconnect", "arn:aws:ecs:us-east-1:123456789:cluster/prod",
		"--yes",
	})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}

	if atomic.LoadInt32(&hits) != 1 {
		t.Errorf("expected 1 server hit, got %d", hits)
	}
	if capturedMethod != http.MethodDelete {
		t.Errorf("method: got %s, want DELETE", capturedMethod)
	}
	if !strings.HasSuffix(capturedPath, "/api/v1/accounts/ecs") {
		t.Errorf("path: got %s", capturedPath)
	}
	if !strings.Contains(capturedQuery, "clusterARN=") {
		t.Errorf("query: got %s, want clusterARN", capturedQuery)
	}

	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("output not JSON: %v — %s", err, stdout.String())
	}
	changed, _ := result["changed"].(bool)
	if !changed {
		t.Errorf("expected changed=true in output, got: %v", result)
	}
}
