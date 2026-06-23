package incidents

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/spf13/cobra"
)

// SetStatusCmd builds `armoctl incidents set-status`.
func SetStatusCmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{
		Use:   "set-status [guid...]",
		Short: "Change the status of runtime incidents (Open|Investigating|Dismissed|Resolved)",
		Long: "Change the status of one or more runtime incidents. Select incidents by GUID " +
			"(positional args and/or --stdin), by --filter key=value, and/or by --search text. " +
			"At least one selection method is required.",
		RunE: func(cmd *cobra.Command, args []string) error {
			statusFlag, _ := cmd.Flags().GetString("status")
			status, err := normalizeStatus(statusFlag)
			if err != nil {
				return err
			}

			guids := append([]string{}, args...)
			if useStdin, _ := cmd.Flags().GetBool("stdin"); useStdin {
				stdinGUIDs, err := readGUIDs(cmd.InOrStdin())
				if err != nil {
					return err
				}
				guids = append(guids, stdinGUIDs...)
			}

			filterPairs, _ := cmd.Flags().GetStringArray("filter")
			filters, err := parseFilters(filterPairs)
			if err != nil {
				return err
			}

			search, _ := cmd.Flags().GetString("search")
			falsePositive, _ := cmd.Flags().GetBool("false-positive")

			return runStatusChange(cmd, clientFor(cmd), statusChangeOpts{
				status:        status,
				guids:         guids,
				filters:       filters,
				searchText:    strings.TrimSpace(search),
				falsePositive: falsePositive,
				commandName:   "incidents.set-status",
			})
		},
	}
	c.Flags().String("status", "", "Target status: Open|Investigating|Dismissed|Resolved (required)")
	c.Flags().StringArray("filter", nil, "Filter as key=value (repeatable); selects incidents to change")
	c.Flags().String("search", "", "Free-text search to select incidents")
	c.Flags().Bool("stdin", false, "Read additional incident GUIDs from stdin")
	c.Flags().Bool("false-positive", false, "Mark the incidents as false positives")
	return c
}

// readGUIDs reads whitespace/newline-separated GUIDs from r.
func readGUIDs(r io.Reader) ([]string, error) {
	var out []string
	sc := bufio.NewScanner(r)
	sc.Split(bufio.ScanWords)
	for sc.Scan() {
		if tok := strings.TrimSpace(sc.Text()); tok != "" {
			out = append(out, tok)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, &clierr.Error{Code: clierr.CodeBadInput, Msg: "failed to read GUIDs from stdin: " + err.Error()}
	}
	return out, nil
}

// parseFilters converts repeated key=value pairs into a single innerFilters map.
func parseFilters(pairs []string) ([]map[string]string, error) {
	if len(pairs) == 0 {
		return nil, nil
	}
	m := map[string]string{}
	for _, p := range pairs {
		k, v, ok := strings.Cut(p, "=")
		k = strings.TrimSpace(k)
		if !ok || k == "" {
			return nil, &clierr.Error{
				Code: clierr.CodeBadInput,
				Msg:  fmt.Sprintf("invalid --filter %q (expected key=value)", p),
			}
		}
		m[k] = strings.TrimSpace(v)
	}
	return []map[string]string{m}, nil
}
