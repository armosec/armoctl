package vulns

import (
	"context"
	"os"

	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/armosec/armoctl/internal/safety"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func ScanCmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{
		Use:   "scan",
		Short: "Trigger a vulnerability scan for the given workload(s)",
		RunE: func(cmd *cobra.Command, args []string) error {
			wlids, _ := cmd.Flags().GetStringSlice("wlid")
			if len(wlids) == 0 {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "scan requires at least one --wlid"}
			}
			body := map[string]any{"wlids": wlids}
			const path = "/vulnerability/scan"

			cli := clientFor(cmd)
			m := cliflags.ReadMutation(cmd)

			return safety.Wrap(cmd.Context(), safety.Args{
				Command: "vulns.scan",
				DryRun:  m.DryRun,
				Yes:     m.Yes,
				Tty:     term.IsTerminal(int(os.Stdin.Fd())),
				Stdout:  cmd.OutOrStdout(),
				Stderr:  cmd.ErrOrStderr(),
				Preview: map[string]any{"method": "POST", "url": path, "body": body},
				ArgsLog: "wlids=" + strjoin(wlids),
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
	c.Flags().StringSlice("wlid", nil, "Workload ID to scan (repeatable)")
	return c
}

func strjoin(s []string) string {
	out := ""
	for i, v := range s {
		if i > 0 {
			out += ","
		}
		out += v
	}
	return out
}
