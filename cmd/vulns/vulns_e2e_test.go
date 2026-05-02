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

// TestE2E_VulnsFlow exercises list/cves-with-filter/severity/scan-dry-run via a
// single shared httptest mux, mirroring the incidents e2e test pattern.
func TestE2E_VulnsFlow(t *testing.T) {
	mux := http.NewServeMux()

	// workloads list
	mux.HandleFunc("/api/v1/vulnerability_v2/workload/list", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("workloads: expected POST, got %s", r.Method)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("workloads: decode body: %v", err)
		}
		if body["pageNum"] == nil {
			t.Errorf("workloads: body missing pageNum: %v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": []map[string]any{
				{"name": "my-deploy", "namespace": "default", "kind": "Deployment", "criticalCount": 3},
			},
			"total": map[string]any{"value": 1},
		})
	})

	// cves list with severity filter
	mux.HandleFunc("/api/v1/vulnerability_v2/vulnerability/list", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("cves: expected POST, got %s", r.Method)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("cves: decode body: %v", err)
		}
		// Verify innerFilters contains severity=Critical
		fl, _ := body["innerFilters"].([]any)
		if len(fl) != 1 {
			t.Errorf("cves: expected 1 innerFilter, got %v", body["innerFilters"])
		} else {
			f, _ := fl[0].(map[string]any)
			if f["severity"] != "Critical" {
				t.Errorf("cves: expected severity=Critical in filter, got: %v", f)
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": []map[string]any{
				{"name": "CVE-2024-9999", "severity": "Critical", "severityScore": 9.8},
			},
			"total": map[string]any{"value": 1},
		})
	})

	// severity aggregate
	mux.HandleFunc("/api/v1/vulnerability/severity", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("severity: expected POST, got %s", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": []map[string]any{
				{"severity": "Critical", "total": 707},
				{"severity": "High", "total": 1234},
			},
			"total": map[string]any{"value": 2},
		})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})

	build := func() (*cobra.Command, *bytes.Buffer) {
		root := &cobra.Command{Use: "armoctl"}
		cliflags.Register(root)
		var stdout bytes.Buffer
		root.SetOut(&stdout)
		root.AddCommand(Cmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
		return root, &stdout
	}

	// vulns workloads → POST list with body pagination → returns 1 mocked item
	t.Run("workloads", func(t *testing.T) {
		root, stdout := build()
		root.SetArgs([]string{"vulns", "workloads"})
		if err := root.ExecuteContext(context.Background()); err != nil {
			t.Fatalf("workloads: %v", err)
		}
		if !strings.Contains(stdout.String(), "my-deploy") {
			t.Fatalf("workloads output missing expected field: %s", stdout.String())
		}
	})

	// vulns cves --severity Critical → POST list with innerFilters → returns 1 mocked CVE
	t.Run("cves_severity_filter", func(t *testing.T) {
		root, stdout := build()
		root.SetArgs([]string{"vulns", "cves", "--severity", "Critical"})
		if err := root.ExecuteContext(context.Background()); err != nil {
			t.Fatalf("cves: %v", err)
		}
		if !strings.Contains(stdout.String(), "CVE-2024-9999") {
			t.Fatalf("cves output missing expected CVE: %s", stdout.String())
		}
	})

	// vulns severity → POST → returns mocked counts
	t.Run("severity", func(t *testing.T) {
		root, stdout := build()
		root.SetArgs([]string{"vulns", "severity"})
		if err := root.ExecuteContext(context.Background()); err != nil {
			t.Fatalf("severity: %v", err)
		}
		if !strings.Contains(stdout.String(), "Critical") {
			t.Fatalf("severity output missing Critical: %s", stdout.String())
		}
	})

	// vulns scan --wlid wlid://test --dry-run → server NOT contacted; output has dryRun: true
	t.Run("scan_dry_run", func(t *testing.T) {
		var serverContacted bool
		dryRunMux := http.NewServeMux()
		dryRunMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			serverContacted = true
			w.WriteHeader(http.StatusOK)
		})
		drySrv := httptest.NewServer(dryRunMux)
		defer drySrv.Close()

		dryC := apiclient.New(apiclient.Config{BaseURL: drySrv.URL, AccessKey: "K", CustomerGUID: "G"})
		root := &cobra.Command{Use: "armoctl"}
		cliflags.Register(root)
		var stdout bytes.Buffer
		root.SetOut(&stdout)
		root.AddCommand(Cmd(func(cmd *cobra.Command) *apiclient.Client { return dryC }))
		root.SetArgs([]string{"vulns", "scan", "--wlid", "wlid://test", "--dry-run"})
		if err := root.ExecuteContext(context.Background()); err != nil {
			t.Fatalf("scan dry-run: %v", err)
		}
		if serverContacted {
			t.Error("scan dry-run: server was contacted, expected no network call")
		}
		if !strings.Contains(stdout.String(), "dryRun") {
			t.Fatalf("scan dry-run output missing dryRun: %s", stdout.String())
		}
	})
}
