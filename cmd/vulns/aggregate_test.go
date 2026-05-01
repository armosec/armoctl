package vulns

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

func TestTop_PostsAndReturnsList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/vulnerability/topVulnerabilities") {
			t.Errorf("path: %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("body decode: %v", err)
		}
		if body == nil {
			t.Error("body is nil")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": []map[string]any{{"name": "CVE-2024-1234", "severity": "High", "count": 42}},
			"total":    map[string]any{"value": 1},
		})
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	root.AddCommand(TopCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))

	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"top"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "CVE-2024-1234") {
		t.Fatalf("output: %s", stdout.String())
	}
}

func TestTop_FiltersBySeverity(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/vulnerability/topVulnerabilities") {
			t.Errorf("path: %s", r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		fl, _ := body["innerFilters"].([]any)
		if len(fl) != 1 {
			t.Errorf("innerFilters: %v", body["innerFilters"])
		} else {
			f, _ := fl[0].(map[string]any)
			if f["severity"] != "Critical" {
				t.Errorf("expected severity=Critical in filter, got: %v", f)
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": []map[string]any{{"name": "CVE-2024-9999", "severity": "Critical"}},
			"total":    map[string]any{"value": 1},
		})
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	root.AddCommand(TopCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))

	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"top", "--severity", "Critical"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "CVE-2024-9999") {
		t.Fatalf("output: %s", stdout.String())
	}
}

func TestSeverity_PostsAndReturnsList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/vulnerability/severity") {
			t.Errorf("path: %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("body decode: %v", err)
		}
		if body == nil {
			t.Error("body is nil")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": []map[string]any{{"severity": "Critical", "total": 707}},
			"total":    map[string]any{"value": 1},
		})
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	root.AddCommand(SeverityCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))

	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"severity"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "Critical") {
		t.Fatalf("output: %s", stdout.String())
	}
}

func TestHistory_PostsAndReturnsList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/vulnerability/overtime") {
			t.Errorf("path: %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("body decode: %v", err)
		}
		if body == nil {
			t.Error("body is nil")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": []map[string]any{{"date": "2024-01-15", "critical": 12, "high": 34}},
			"total":    map[string]any{"value": 1},
		})
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	root.AddCommand(HistoryCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))

	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"history"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "2024-01-15") {
		t.Fatalf("output: %s", stdout.String())
	}
}

func TestScan_DryRun(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	root.AddCommand(ScanCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))

	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"scan", "--wlid", "wlid://cluster-foo/ns/default/deploy/my-app", "--dry-run"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}

	if atomic.LoadInt32(&hits) != 0 {
		t.Errorf("server was contacted during dry-run (%d hits)", hits)
	}

	out := stdout.String()
	if !strings.Contains(out, "dryRun") {
		t.Fatalf("output missing dryRun: %s", out)
	}
	if !strings.Contains(out, "my-app") {
		t.Fatalf("output missing wlid in preview: %s", out)
	}
}

func TestScan_Yes(t *testing.T) {
	var hits int32
	var capturedBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		if !strings.HasSuffix(r.URL.Path, "/vulnerability/scan") {
			t.Errorf("path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("method: %s", r.Method)
		}
		_ = json.NewDecoder(r.Body).Decode(&capturedBody)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "triggered"})
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	root.AddCommand(ScanCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))

	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"scan", "--wlid", "wlid://cluster-foo/ns/default/deploy/my-app", "--yes"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}

	if atomic.LoadInt32(&hits) != 1 {
		t.Errorf("expected 1 server hit, got %d", hits)
	}

	wlids, _ := capturedBody["wlids"].([]any)
	if len(wlids) != 1 || wlids[0] != "wlid://cluster-foo/ns/default/deploy/my-app" {
		t.Errorf("unexpected wlids in request body: %v", capturedBody["wlids"])
	}

	out := stdout.String()
	if !strings.Contains(out, "changed") {
		t.Fatalf("output missing changed: %s", out)
	}
}
