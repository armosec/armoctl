package incidents

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

// TestE2E_TriageFlow exercises list → resolve --dry-run → resolve --yes.
func TestE2E_TriageFlow(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/runtime/incidents", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("incidents list: expected POST, got %s", r.Method)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if _, ok := body["pageNum"]; !ok {
			t.Errorf("body missing pageNum: %v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": []map[string]any{{"guid": "i1", "name": "X", "kind": "ThreatDetection", "attributes": map[string]any{"incidentStatus": "open"}, "noise": "n"}},
			"total":    map[string]any{"value": 1},
		})
	})
	mux.HandleFunc("/api/v1/runtime/incidents/i1/resolve", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	t.Setenv("ARMOCTL_AUDIT_LOG", t.TempDir()+"/audit.log")
	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})

	build := func() *cobra.Command {
		root := &cobra.Command{Use: "armoctl"}
		cliflags.Register(root)
		root.AddCommand(Cmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
		return root
	}

	// list
	{
		root := build()
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetArgs([]string{"incidents", "list"})
		if err := root.ExecuteContext(context.Background()); err != nil {
			t.Fatalf("list: %v", err)
		}
		if !strings.Contains(out.String(), "i1") || strings.Contains(out.String(), "noise") {
			t.Fatalf("list output: %s", out.String())
		}
	}
	// resolve --dry-run
	{
		root := build()
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetArgs([]string{"incidents", "resolve", "i1", "--reason", "fp", "--dry-run"})
		if err := root.ExecuteContext(context.Background()); err != nil {
			t.Fatalf("dry-run: %v", err)
		}
		if !strings.Contains(out.String(), `"dryRun"`) {
			t.Fatalf("dry-run output: %s", out.String())
		}
	}
	// resolve --yes
	{
		root := build()
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetArgs([]string{"incidents", "resolve", "i1", "--reason", "fp", "--yes"})
		if err := root.ExecuteContext(context.Background()); err != nil {
			t.Fatalf("yes: %v", err)
		}
		if !strings.Contains(out.String(), `"changed": true`) {
			t.Fatalf("yes output: %s", out.String())
		}
	}
}
