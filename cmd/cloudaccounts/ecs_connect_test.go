package cloudaccounts

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

func TestECSConnect_FetchesLink(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method: got %s, want GET", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/accounts/ecs") {
			t.Errorf("path: got %s", r.URL.Path)
		}
		if !strings.Contains(r.URL.RawQuery, "clusterARN=") {
			t.Errorf("query missing clusterARN: %s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"cloudFormationUrl": "https://console.aws.amazon.com/cloudformation/...",
			"clusterARN":        "arn:aws:ecs:us-east-1:123456789:cluster/prod",
		})
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.AddCommand(ECSConnectCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))
	root.SetArgs([]string{"connect", "arn:aws:ecs:us-east-1:123456789:cluster/prod"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "cloudFormationUrl") {
		t.Fatalf("output missing cloudFormationUrl field: %s", stdout.String())
	}
}
