package risks

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

func newExcRoot() (*cobra.Command, *bytes.Buffer) {
	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	return root, &stdout
}

func TestExceptionsList_PostsListWithPagination(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: got %s, want POST", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/securityrisks/exceptions/list") {
			t.Errorf("path: got %s", r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["pageNum"] == nil || body["pageSize"] == nil {
			t.Errorf("body missing pageNum/pageSize: %v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": []map[string]any{
				{"guid": "e1", "name": "accept-risk-1", "policyIDs": []string{"R-1"}},
				{"guid": "e2", "name": "accept-risk-2", "policyIDs": []string{"R-2"}},
			},
			"total": map[string]any{"value": 2},
		})
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root, stdout := newExcRoot()
	exc := &cobra.Command{Use: "exceptions"}
	exc.AddCommand(ExceptionsListCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	root.AddCommand(exc)
	root.SetArgs([]string{"exceptions", "list"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	out := stdout.String()
	if !strings.Contains(out, `"items"`) || !strings.Contains(out, "e1") || !strings.Contains(out, "e2") {
		t.Fatalf("unexpected list output: %s", out)
	}
}
