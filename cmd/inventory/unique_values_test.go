package inventory

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

func TestUniqueValues_PostsWithRightBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: got %s, want %s", r.Method, http.MethodPost)
		}
		if !strings.HasSuffix(r.URL.Path, "/uniqueValues/inventory") {
			t.Errorf("path: got %s, want suffix /uniqueValues/inventory", r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		fields, ok := body["fields"].(map[string]any)
		if !ok || fields == nil {
			t.Errorf("body missing or malformed fields: %v", body)
		}
		_, hasField := fields["cluster"]
		if !hasField {
			t.Errorf("body missing cluster field: %v", fields)
		}
		resp := map[string]any{
			"cluster": []string{"cluster-1", "cluster-2"},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	root.AddCommand(UniqueValuesCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))

	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"unique-values", "cluster"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "cluster-1") {
		t.Fatalf("output missing expected value: %s", stdout.String())
	}
}
