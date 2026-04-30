package incidents

import (
	"context"

	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/armosec/armoctl/internal/safety"
	"github.com/spf13/cobra"
)

func ResolveCmd(clientFor ClientFor) *cobra.Command {
	c := &cobra.Command{
		Use:   "resolve [guid]",
		Short: "Resolve a runtime incident (sets status to Resolved)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "resolve requires a GUID"}
			}
			falsePositive, _ := cmd.Flags().GetBool("false-positive")
			body := map[string]any{
				"status":                "Resolved",
				"incidentsGuids":        []string{args[0]},
				"innerFilters":          []any{},
				"markedAsFalsePositive": falsePositive,
			}
			const path = "/runtime/incidents/changeStatus"

			cli := clientFor(cmd)
			m := cliflags.ReadMutation(cmd)

			return safety.Wrap(cmd.Context(), safety.Args{
				Command: "incidents.resolve",
				DryRun:  m.DryRun,
				Yes:     m.Yes,
				Tty:     false,
				Stdout:  cmd.OutOrStdout(),
				Stderr:  cmd.ErrOrStderr(),
				Preview: map[string]any{"method": "POST", "url": path, "body": body},
				ArgsLog: "guid=" + args[0],
				Exec: func(ctx context.Context) (any, safety.ExecMeta, error) {
					var resp map[string]any
					err := cli.PostJSON(ctx, path, nil, body, &resp)
					if err != nil {
						return nil, safety.ExecMeta{}, err
					}
					return resp, safety.ExecMeta{URL: "POST " + path, Status: 200}, nil
				},
			})
		},
	}
	c.Flags().Bool("false-positive", false, "Mark the incident as a false positive when resolving")
	return c
}
