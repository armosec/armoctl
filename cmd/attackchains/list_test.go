package attackchains

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

func newRoot(clientFor func(*cobra.Command) *apiclient.Client) (*cobra.Command, *bytes.Buffer) {
	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	return root, &stdout
}

func TestList_PostsList(t *testing.T) {
	// The live endpoint returns { "response": { "attackChains": [...] }, "total": {...} }
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: got %s, want %s", r.Method, http.MethodPost)
		}
		if !strings.HasSuffix(r.URL.Path, "/attackchains") {
			t.Errorf("path: got %s, want suffix /attackchains", r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["pageNum"] == nil || body["pageSize"] == nil {
			t.Errorf("body missing pageNum/pageSize: %v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": map[string]any{
				"attackChains": []map[string]any{{"name": "chain-1", "guid": "guid-123", "severity": "critical"}},
			},
			"total": map[string]any{"value": 1},
		})
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root, stdout := newRoot(nil)
	root.AddCommand(ListCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	root.SetArgs([]string{"list"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "chain-1") {
		t.Fatalf("output missing expected item: %s", stdout.String())
	}
}
