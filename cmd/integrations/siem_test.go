package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/spf13/cobra"
)

func TestSiemCreate_DryRun(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "splunk-config.json")
	configJSON := `{"host": "splunk.example.com", "token": "abc123"}`
	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatal(err)
	}

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	var stdout bytes.Buffer
	root.SetOut(&stdout)

	integ := &cobra.Command{Use: "integrations"}
	siem := &cobra.Command{Use: "siem"}
	siem.AddCommand(SiemCreateCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	integ.AddCommand(siem)
	root.AddCommand(integ)

	root.SetArgs([]string{
		"integrations", "siem", "create", "splunk",
		"--config-file", configPath,
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
		t.Fatalf("output not JSON: %v", err)
	}

	dryRun, _ := result["dryRun"].(bool)
	if !dryRun {
		t.Errorf("expected dryRun=true in output")
	}
}

func TestSiemCreate_Yes(t *testing.T) {
	var capturedMethod string
	var capturedPath string
	var capturedBody map[string]any
	var hits int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&capturedBody)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "created"})
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "splunk-config.json")
	configJSON := `{"host": "splunk.example.com", "token": "abc123", "index": "main"}`
	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatal(err)
	}

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	var stdout bytes.Buffer
	root.SetOut(&stdout)

	integ := &cobra.Command{Use: "integrations"}
	siem := &cobra.Command{Use: "siem"}
	siem.AddCommand(SiemCreateCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	integ.AddCommand(siem)
	root.AddCommand(integ)

	root.SetArgs([]string{
		"integrations", "siem", "create", "splunk",
		"--config-file", configPath,
		"--yes",
	})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}

	if atomic.LoadInt32(&hits) != 1 {
		t.Errorf("expected 1 server hit, got %d", hits)
	}
	if capturedMethod != http.MethodPost {
		t.Errorf("method: got %s, want POST", capturedMethod)
	}
	if !strings.HasSuffix(capturedPath, "/siem/splunk") {
		t.Errorf("path: got %s", capturedPath)
	}
	if capturedBody["host"] != "splunk.example.com" {
		t.Errorf("host: got %v", capturedBody["host"])
	}
	if capturedBody["token"] != "abc123" {
		t.Errorf("token: got %v", capturedBody["token"])
	}
	if capturedBody["index"] != "main" {
		t.Errorf("index: got %v", capturedBody["index"])
	}

	var result map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("output not JSON: %v", err)
	}
	changed, _ := result["changed"].(bool)
	if !changed {
		t.Errorf("expected changed=true in output")
	}
}
