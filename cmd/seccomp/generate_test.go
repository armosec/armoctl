package seccomp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/spf13/cobra"
)

func newGenRoot(clientFor func(*cobra.Command) *apiclient.Client) (*cobra.Command, *bytes.Buffer) {
	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	return root, &stdout
}

func TestGenerate_DryRun(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root, stdout := newGenRoot(nil)
	root.AddCommand(GenerateCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	root.SetArgs([]string{
		"generate",
		"--wlid", "wl-456",
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
	body, _ := req["body"].(map[string]any)
	if body == nil {
		t.Fatalf("body missing in request: %v", req)
	}

	wlids, _ := body["wlids"].([]any)
	if len(wlids) != 1 || wlids[0] != "wl-456" {
		t.Errorf("wlids: got %v", body["wlids"])
	}
}

func TestGenerate_Yes(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		if r.Method != http.MethodPost {
			t.Errorf("method: got %s, want POST", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/seccomp/generate") {
			t.Errorf("path: got %s", r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		_ = json.NewEncoder(w).Encode(map[string]any{"changed": true, "profiles": []map[string]string{}})
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root, stdout := newGenRoot(nil)
	root.AddCommand(GenerateCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	root.SetArgs([]string{
		"generate",
		"--wlid", "wl-456",
		"--yes",
	})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}

	if atomic.LoadInt32(&hits) != 1 {
		t.Errorf("expected 1 server hit, got %d", hits)
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

func TestGenerate_NoWlidFails(t *testing.T) {
	c := apiclient.New(apiclient.Config{BaseURL: "http://localhost", AccessKey: "K", CustomerGUID: "G"})
	root, _ := newGenRoot(nil)
	root.AddCommand(GenerateCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	root.SetArgs([]string{
		"generate",
		"--yes",
	})
	err := root.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected error when no --wlid provided, got nil")
	}
	// The error from cobra's required flag is not a *clierr.Error,
	// but we can verify that the command rejected the input
	if !strings.Contains(err.Error(), "wlid") {
		t.Errorf("error should mention wlid flag, got: %v", err)
	}
}

func isCliErr(err error, target **clierr.Error) bool {
	return err != nil && errors.As(err, target)
}
