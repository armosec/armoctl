package runtimerules

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/spf13/cobra"
)

func TestEvaluate_PostsBodyWithRuleAndInput(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/runtime/rules/evaluate" {
			t.Errorf("path: %s", r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if _, ok := body["rule"]; !ok {
			t.Errorf("body missing rule: %v", body)
		}
		if _, ok := body["input"]; !ok {
			t.Errorf("body missing input: %v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"matches": true,
			"reason":  "rule matched input",
		})
	}))
	defer srv.Close()

	c := apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})

	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	root.AddCommand(Cmd(func(cmd *cobra.Command) *apiclient.Client { return c }))

	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stdout)
	root.SetArgs([]string{"runtime-rules", "evaluate", "--rule", `{"condition":"true"}`, "--input", `{"data":"test"}`})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
}
