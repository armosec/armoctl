package networkpolicies

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestFields_NotEmpty(t *testing.T) {
	cs := Cheatsheet()
	if len(cs) == 0 {
		t.Fatal("cheatsheet is empty")
	}
}

func TestFields_Cmd(t *testing.T) {
	root := &cobra.Command{Use: "armoctl"}
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.AddCommand(FieldsCmd())
	root.SetArgs([]string{"fields"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}
	output := stdout.String()
	if !strings.Contains(output, "policies") {
		t.Fatalf("output missing 'policies' scope: %s", output)
	}
	if !strings.Contains(output, "name") {
		t.Fatalf("output missing 'name' field: %s", output)
	}
}
