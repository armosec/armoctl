package networkpolicies

import (
	"context"
	"os"
	"strings"

	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/armosec/armoctl/internal/safety"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func GenerateCmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{
		Use:   "generate",
		Short: "Generate network policies for specified workloads",
		RunE: func(cmd *cobra.Command, args []string) error {
			wlids, _ := cmd.Flags().GetStringSlice("wlid")
			if len(wlids) == 0 {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "generate requires at least one --wlid"}
			}

			cli := clientFor(cmd)
			m := cliflags.ReadMutation(cmd)
			path := "/networkpolicies/generate"
			body := map[string]any{"wlids": wlids}

			return safety.Wrap(cmd.Context(), safety.Args{
				Command: "networkpolicies.generate",
				DryRun:  m.DryRun,
				Yes:     m.Yes,
				Tty:     term.IsTerminal(int(os.Stdin.Fd())),
				Stdout:  cmd.OutOrStdout(),
				Stderr:  cmd.ErrOrStderr(),
				Preview: map[string]any{"method": "POST", "url": path, "body": body},
				ArgsLog: "wlids=" + strings.Join(wlids, ","),
				Exec: func(ctx context.Context) (any, safety.ExecMeta, error) {
					var resp map[string]any
					if err := cli.PostJSON(ctx, path, nil, body, &resp); err != nil {
						return nil, safety.ExecMeta{}, err
					}
					return resp, safety.ExecMeta{URL: "POST " + path, Status: 200}, nil
				},
			})
		},
	}
	c.Flags().StringSlice("wlid", nil, "Workload ID (repeatable, required)")
	return c
}
