// Package cliflags registers the persistent flags shared across every
// armoctl resource command and exposes a single Resolve() to read them.
package cliflags

import (
	"strings"

	"github.com/armosec/armoctl/internal/output"
	"github.com/spf13/cobra"
)

// Register installs the persistent flags on root.
func Register(root *cobra.Command) {
	pf := root.PersistentFlags()
	pf.String("output", "json", "Output format: json|yaml|ndjson|table|csv")
	pf.String("query", "", "gojq expression applied after --fields/summary")
	pf.StringSlice("fields", nil, "Comma-separated dotted paths to keep")
	pf.Bool("full", false, "Disable summary projection; return raw response")
	pf.Int("limit", 500, "Max items to fetch when auto-paging (0 = no cap, requires --yes)")
	pf.Int("page", 0, "Explicit page (1-based; disables auto-paging)")
	pf.Int("page-size", 0, "Explicit page size (disables auto-paging)")
	pf.Bool("dry-run", false, "Build the request but do not send it")
	pf.Bool("yes", false, "Skip confirmation for mutations")
}

// OutputOptions reads the output-related flags from cmd.
func OutputOptions(cmd *cobra.Command, summary []string) output.Options {
	o := output.Options{
		Format:        flagString(cmd, "output"),
		Query:         flagString(cmd, "query"),
		Full:          flagBool(cmd, "full"),
		SummaryFields: summary,
	}
	if raw := flagString(cmd, "fields"); raw != "" {
		o.Fields = splitFields(raw)
	} else if fs, _ := cmd.Flags().GetStringSlice("fields"); len(fs) > 0 {
		o.Fields = fs
	}
	return o
}

func splitFields(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// PageOptions reads the pagination flags.
type PageOptions struct {
	Limit    int
	Page     int
	PageSize int
}

func ReadPage(cmd *cobra.Command) PageOptions {
	return PageOptions{
		Limit:    flagInt(cmd, "limit"),
		Page:     flagInt(cmd, "page"),
		PageSize: flagInt(cmd, "page-size"),
	}
}

// MutationOptions reads dry-run / yes.
type MutationOptions struct {
	DryRun bool
	Yes    bool
}

func ReadMutation(cmd *cobra.Command) MutationOptions {
	return MutationOptions{DryRun: flagBool(cmd, "dry-run"), Yes: flagBool(cmd, "yes")}
}

func flagString(cmd *cobra.Command, n string) string {
	v, _ := cmd.Flags().GetString(n)
	return v
}
func flagBool(cmd *cobra.Command, n string) bool { v, _ := cmd.Flags().GetBool(n); return v }
func flagInt(cmd *cobra.Command, n string) int   { v, _ := cmd.Flags().GetInt(n); return v }
