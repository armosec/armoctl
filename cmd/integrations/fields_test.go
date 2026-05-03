package integrations

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/spf13/cobra"
)

func newIntRoot(clientFor func(*cobra.Command) *apiclient.Client) (*cobra.Command, *bytes.Buffer) {
	root := &cobra.Command{Use: "armoctl"}
	cliflags.Register(root)
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	return root, &stdout
}

func TestFieldsCmd(t *testing.T) {
	c := apiclient.New(apiclient.Config{BaseURL: "http://localhost", AccessKey: "K", CustomerGUID: "G"})
	root, stdout := newIntRoot(nil)
	integ := &cobra.Command{Use: "integrations"}
	integ.AddCommand(FieldsCmd())
	root.AddCommand(integ)
	root.SetArgs([]string{"integrations", "fields"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}

	output := stdout.String()
	if !strings.Contains(output, "jira-projects") {
		t.Errorf("expected 'jira-projects' in output")
	}
	if !strings.Contains(output, "jira-issue-types") {
		t.Errorf("expected 'jira-issue-types' in output")
	}
	if !strings.Contains(output, "Project key") {
		t.Errorf("expected field description in output")
	}
	_ = c
}
