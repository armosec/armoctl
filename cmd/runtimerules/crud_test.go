package runtimerules

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

func TestCreate_DryRunDoesNotCallServer(t *testing.T) {
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
	root.AddCommand(Cmd(func(cmd *cobra.Command) *apiclient.Client { return c }))

	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"runtime-rules", "create", "--name", "test-rule", "--rule", `{}`, "--dry-run"})
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

func TestCreate_YesPostsAndReports(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method: %s", r.Method)
		}
		if r.URL.Path != "/api/v1/runtime/rules" {
			t.Errorf("path: %s", r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "test-rule" {
			t.Errorf("body.name: %v", body["name"])
		}
		if body["type"] != "Custom" {
			t.Errorf("body.type: %v", body["type"])
		}
		w.Header().Set("x-request-id", "req-z")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"guid":"r1"}`))
	}))
	defer srv.Close()
	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})

	t.Setenv("ARMOCTL_AUDIT_LOG", t.TempDir()+"/audit.log")

	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	root.AddCommand(Cmd(func(cmd *cobra.Command) *apiclient.Client { return c }))

	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"runtime-rules", "create", "--name", "test-rule", "--rule", `{}`, "--yes"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), `"changed": true`) {
		t.Fatalf("changed not set: %s", stdout.String())
	}
}

func TestCreate_RequiresNameAndRule(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()
	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})

	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	root.AddCommand(Cmd(func(cmd *cobra.Command) *apiclient.Client { return c }))

	tests := []struct {
		name string
		args []string
	}{
		{"missing name", []string{"runtime-rules", "create", "--rule", "{}"}},
		{"missing rule", []string{"runtime-rules", "create", "--name", "test"}},
		{"empty", []string{"runtime-rules", "create"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			root.SetOut(&stdout)
			root.SetErr(&stdout)
			root.SetArgs(tt.args)
			if err := root.ExecuteContext(context.Background()); err == nil {
				t.Errorf("expected error but got none")
			}
		})
	}
}

func TestUpdate_DryRunShowsPUT(t *testing.T) {
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
	root.AddCommand(Cmd(func(cmd *cobra.Command) *apiclient.Client { return c }))

	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"runtime-rules", "update", "--guid", "r1", "--name", "updated", "--dry-run"})
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
	req, _ := got["request"].(map[string]any)
	if req == nil || req["method"] != "PUT" {
		t.Fatalf("expected PUT method: %v", got)
	}
	if req["url"] != "/runtime/rules" {
		t.Fatalf("expected /runtime/rules path: %v", req)
	}
	body, _ := req["body"].(map[string]any)
	if body["guid"] != "r1" {
		t.Fatalf("body should have guid: %v", body)
	}
}

func TestDelete_YesCallsDELETE(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/runtime/rules/r1" {
			t.Errorf("path: %s", r.URL.Path)
		}
		w.Header().Set("x-request-id", "req-z")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})

	t.Setenv("ARMOCTL_AUDIT_LOG", t.TempDir()+"/audit.log")

	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	root.AddCommand(Cmd(func(cmd *cobra.Command) *apiclient.Client { return c }))

	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"runtime-rules", "delete", "r1", "--yes"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), `"changed": true`) {
		t.Fatalf("changed not set: %s", stdout.String())
	}
}
