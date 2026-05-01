// Package schema embeds JSON schemas for armoctl resources and provides
// the `armoctl schema` cobra command.
package schema

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/armosec/armoctl/internal/clierr"
	"github.com/spf13/cobra"
)

//go:embed data/*.json
var fsys embed.FS

// List returns the resource names whose schemas are embedded.
func List() []string {
	var names []string
	_ = fs.WalkDir(fsys, "data", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		names = append(names, strings.TrimSuffix(strings.TrimPrefix(p, "data/"), ".json"))
		return nil
	})
	sort.Strings(names)
	return names
}

// Get returns the JSON schema bytes for resource.
func Get(resource string) ([]byte, error) {
	b, err := fsys.ReadFile("data/" + resource + ".json")
	if err != nil {
		return nil, &clierr.Error{Code: clierr.CodeNotFound, Msg: fmt.Sprintf("no schema for %q", resource)}
	}
	// validate parseable
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, fmt.Errorf("schema for %s is not valid JSON: %w", resource, err)
	}
	return b, nil
}

// Cmd builds the `armoctl schema` cobra command.
func Cmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "schema [resource]",
		Short: "Print JSON schemas for armoctl resources",
		RunE: func(cmd *cobra.Command, args []string) error {
			list, _ := cmd.Flags().GetBool("list")
			if list {
				for _, n := range List() {
					_, _ = fmt.Fprintln(cmd.OutOrStdout(), n)
				}
				return nil
			}
			if len(args) != 1 {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "schema requires a resource name (or --list)"}
			}
			b, err := Get(args[0])
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(b))
			return nil
		},
	}
	c.Flags().Bool("list", false, "List embedded schema resource names")
	return c
}
