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

func TestAlerts_ListsAlertsForIncident(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/runtime/incidents/i1/alerts/list") {
			t.Errorf("path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": []map[string]any{{"alertID": "a1"}},
			"total":    map[string]any{"value": 1},
		})
	}))
	defer srv.Close()
	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})

	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	root.AddCommand(AlertsCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))

	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"alerts", "i1", "--full"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "a1") {
		t.Fatalf("output: %s", stdout.String())
	}
}
