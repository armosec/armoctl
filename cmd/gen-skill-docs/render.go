package main

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/armosec/armoctl/internal/skillmeta"
)

// renderSkill returns the SKILL.md byte content for the given cluster.
// It is a pure function: same inputs → same output, no I/O.
func renderSkill(m skillmeta.Meta, clusterCmd *cobra.Command) []byte {
	var b bytes.Buffer

	// Frontmatter
	fmt.Fprintln(&b, "---")
	fmt.Fprintf(&b, "name: %s\n", m.Name)
	fmt.Fprintf(&b, "description: %s\n", oneLine(m.Description))
	fmt.Fprintln(&b, "---")
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "# %s\n\n", m.Name)
	fmt.Fprintf(&b, "%s\n\n", m.Summary)

	// Commands
	fmt.Fprintln(&b, "## Commands")
	fmt.Fprintln(&b)
	if clusterCmd != nil {
		writeCommandTree(&b, clusterCmd, "")
	}
	fmt.Fprintln(&b)

	// Resource fields
	if len(m.Cheatsheet) > 0 {
		fmt.Fprintln(&b, "## Resource fields")
		fmt.Fprintln(&b)
		scopes := make([]string, 0, len(m.Cheatsheet))
		for s := range m.Cheatsheet {
			scopes = append(scopes, s)
		}
		sort.Strings(scopes)
		for _, s := range scopes {
			fmt.Fprintf(&b, "### %s\n\n", s)
			fmt.Fprintln(&b, "| Field | Description |")
			fmt.Fprintln(&b, "|---|---|")
			for _, f := range m.Cheatsheet[s] {
				fmt.Fprintf(&b, "| `%s` | %s |\n", f.Name, escapePipes(f.Doc))
			}
			fmt.Fprintln(&b)
		}
	}

	// Field semantics
	if len(m.FieldNotes) > 0 {
		fmt.Fprintln(&b, "## Field semantics")
		fmt.Fprintln(&b)
		names := make([]string, 0, len(m.FieldNotes))
		for n := range m.FieldNotes {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			fmt.Fprintf(&b, "**`%s`** — %s\n\n", n, m.FieldNotes[n])
		}
	}

	// Recipes
	if len(m.Recipes) > 0 {
		fmt.Fprintln(&b, "## Recipes")
		fmt.Fprintln(&b)
		for _, r := range m.Recipes {
			fmt.Fprintf(&b, "### %s\n\n", r.Title)
			fmt.Fprintf(&b, "%s\n\n", r.Body)
		}
	}

	return b.Bytes()
}

func writeCommandTree(b *bytes.Buffer, cmd *cobra.Command, prefix string) {
	for _, sub := range cmd.Commands() {
		if sub.Hidden {
			continue
		}
		full := strings.TrimSpace(prefix + " " + sub.Use)
		fmt.Fprintf(b, "- `%s` — %s\n", full, sub.Short)
		if hasFlags(sub) {
			fmt.Fprintln(b, "  - Flags:")
			sub.LocalFlags().VisitAll(func(f *pflagFlag) {
				fmt.Fprintf(b, "    - `--%s` (%s) %s\n", f.Name, f.Value.Type(), f.Usage)
			})
		}
		if len(sub.Commands()) > 0 {
			writeCommandTree(b, sub, full)
		}
	}
}

func hasFlags(c *cobra.Command) bool {
	any := false
	c.LocalFlags().VisitAll(func(*pflagFlag) { any = true })
	return any
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func escapePipes(s string) string {
	return strings.ReplaceAll(s, "|", `\|`)
}

func main() {} // replaced in Task 5
