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

func TestList_PrintsItemsAsJSONList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if _, ok := body["pageNum"]; !ok {
			t.Errorf("body missing pageNum: %v", body)
		}
		if _, ok := body["pageSize"]; !ok {
			t.Errorf("body missing pageSize: %v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": []map[string]any{
				{"guid": "i1", "name": "Suspicious binary", "attributes": map[string]any{"incidentStatus": "open"}, "kind": "ThreatDetection", "noise": "x"},
				{"guid": "i2", "name": "C2 beacon", "attributes": map[string]any{"incidentStatus": "open"}, "kind": "ThreatDetection", "noise": "y"},
			},
			"total": map[string]any{"value": 2},
		})
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})

	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	root.AddCommand(ListCmd(func(cmd *cobra.Command) *apiclient.Client { return c }))

	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stdout)
	root.SetArgs([]string{"list"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	out := stdout.String()
	if !strings.Contains(out, `"items"`) || !strings.Contains(out, "i1") || !strings.Contains(out, "i2") {
		t.Fatalf("unexpected list output: %s", out)
	}
	if strings.Contains(out, "noise") {
		t.Fatalf("default summary should drop noise: %s", out)
	}
}
