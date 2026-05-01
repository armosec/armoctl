package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/spf13/cobra"
)

func TestAlertChannels_DryRun(t *testing.T) {
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

	integ := &cobra.Command{Use: "integrations"}
	ac := &cobra.Command{Use: "alert-channels"}
	ac.AddCommand(AlertChannelsCreateCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	integ.AddCommand(ac)
	root.AddCommand(integ)

	root.SetArgs([]string{
		"integrations", "alert-channels", "create", "test-guid-123",
		"--type", "slack",
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

func TestAlertChannels_Yes(t *testing.T) {
	var capturedMethod string
	var capturedPath string
	var capturedBody map[string]any
	var hits int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&capturedBody)
		_ = json.NewEncoder(w).Encode(map[string]any{"guid": "test-guid-123", "type": "slack"})
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	var stdout bytes.Buffer
	root.SetOut(&stdout)

	integ := &cobra.Command{Use: "integrations"}
	ac := &cobra.Command{Use: "alert-channels"}
	ac.AddCommand(AlertChannelsCreateCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	integ.AddCommand(ac)
	root.AddCommand(integ)

	root.SetArgs([]string{
		"integrations", "alert-channels", "create", "test-guid-123",
		"--type", "slack",
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
	if capturedPath != "/api/v1/notifications/alertChannel/test-guid-123" {
		t.Errorf("path: got %s", capturedPath)
	}
	if capturedBody["type"] != "slack" {
		t.Errorf("type: got %v, want slack", capturedBody["type"])
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

func TestAlertChannels_WithConfigFile(t *testing.T) {
	var capturedBody map[string]any
	var hits int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		_ = json.NewDecoder(r.Body).Decode(&capturedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "slack-config.json")
	configJSON := `{"webhook_url": "https://hooks.slack.com/test", "channel": "#alerts"}`
	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatal(err)
	}

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	var stdout bytes.Buffer
	root.SetOut(&stdout)

	integ := &cobra.Command{Use: "integrations"}
	ac := &cobra.Command{Use: "alert-channels"}
	ac.AddCommand(AlertChannelsCreateCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	integ.AddCommand(ac)
	root.AddCommand(integ)

	root.SetArgs([]string{
		"integrations", "alert-channels", "create", "test-guid",
		"--type", "slack",
		"--config-file", configPath,
		"--yes",
	})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}

	if atomic.LoadInt32(&hits) != 1 {
		t.Errorf("expected 1 server hit, got %d", hits)
	}
	if capturedBody["type"] != "slack" {
		t.Errorf("type: got %v, want slack", capturedBody["type"])
	}
	if capturedBody["webhook_url"] != "https://hooks.slack.com/test" {
		t.Errorf("webhook_url: got %v", capturedBody["webhook_url"])
	}
	if capturedBody["channel"] != "#alerts" {
		t.Errorf("channel: got %v", capturedBody["channel"])
	}
}
