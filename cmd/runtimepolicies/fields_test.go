package runtimepolicies

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestFieldsCmd(t *testing.T) {
	root := &cobra.Command{Use: "armoctl"}
	root.AddCommand(FieldsCmd())

	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"fields"})
	if err := root.ExecuteContext(context.Background()); err != nil {
		t.Fatal(err)
	}

	out := stdout.String()
	expectedFields := []string{"guid", "name", "description", "enabled", "scope", "creationTime"}
	for _, field := range expectedFields {
		if !strings.Contains(out, field) {
			t.Errorf("expected field %q in output: %s", field, out)
		}
	}
}
