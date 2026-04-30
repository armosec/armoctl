package incidents

import (
	"context"

	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/armosec/armoctl/internal/safety"
	"github.com/spf13/cobra"
)

func UnresolveCmd(clientFor ClientFor) *cobra.Command {
	return &cobra.Command{
		Use:   "unresolve [guid]",
		Short: "Reopen a previously-resolved incident",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "unresolve requires a GUID"}
			}
			path := "/runtime/incidents/" + args[0] + "/unresolve"
			cli := clientFor(cmd)
			m := cliflags.ReadMutation(cmd)
			return safety.Wrap(cmd.Context(), safety.Args{
				Command: "incidents.unresolve",
				DryRun:  m.DryRun,
				Yes:     m.Yes,
				Tty:     false,
				Stdout:  cmd.OutOrStdout(),
				Stderr:  cmd.ErrOrStderr(),
				Preview: map[string]any{"method": "POST", "url": path},
				ArgsLog: "guid=" + args[0],
				Exec: func(ctx context.Context) (any, safety.ExecMeta, error) {
					var resp map[string]any
					if err := cli.PostJSON(ctx, path, nil, nil, &resp); err != nil {
						return nil, safety.ExecMeta{}, err
					}
					return resp, safety.ExecMeta{URL: "POST " + path, Status: 200}, nil
				},
			})
		},
	}
}
