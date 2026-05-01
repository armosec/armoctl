package posture

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/armosec/armoctl/internal/safety"
	"github.com/spf13/cobra"
)

func ExceptionsDeleteCmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{
		Use:   "delete [policy-name]",
		Short: "Delete a posture exception policy by name",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "delete requires the policy name"}
			}
			cli := clientFor(cmd)
			m := cliflags.ReadMutation(cmd)
			path := "/postureExceptionPolicy"
			query := url.Values{"policyName": {args[0]}}

			return safety.Wrap(cmd.Context(), safety.Args{
				Command: "posture.exceptions.delete",
				DryRun:  m.DryRun, Yes: m.Yes, Tty: false,
				Stdout: cmd.OutOrStdout(), Stderr: cmd.ErrOrStderr(),
				Preview: map[string]any{"method": "DELETE", "url": path + "?policyName=" + args[0]},
				ArgsLog: "policyName=" + args[0],
				Exec: func(ctx context.Context) (any, safety.ExecMeta, error) {
					resp, err := cli.Do(ctx, http.MethodDelete, path, query, nil)
					if err != nil {
						return nil, safety.ExecMeta{}, err
					}
					defer resp.Body.Close()
					if resp.StatusCode >= 400 {
						b, _ := io.ReadAll(resp.Body)
						return nil, safety.ExecMeta{}, &clierr.Error{Code: clierr.CodeServer, Msg: strings.TrimSpace(string(b))}
					}
					return map[string]any{"deleted": args[0]}, safety.ExecMeta{URL: "DELETE " + path, Status: resp.StatusCode}, nil
				},
			})
		},
	}
	return c
}
