package cliflags

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestRegisterAddsExpectedFlags(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	Register(root)
	for _, name := range []string{"output", "query", "fields", "full", "limit", "page", "page-size", "dry-run", "yes"} {
		if root.PersistentFlags().Lookup(name) == nil {
			t.Errorf("flag %q not registered", name)
		}
	}
}
