package cloudaccounts

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/armosec/armoctl/internal/safety"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func ECSDisconnectCmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{
		Use:   "disconnect [cluster-arn]",
		Short: "Disconnect an ECS cluster from ARMO",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "disconnect requires the cluster ARN"}
			}
			clusterARN := args[0]
			cli := clientFor(cmd)
			m := cliflags.ReadMutation(cmd)
			path := "/accounts/ecs"
			query := url.Values{"clusterARN": {clusterARN}}

			return safety.Wrap(cmd.Context(), safety.Args{
				Command: "cloudaccounts.ecs.disconnect",
				DryRun:  m.DryRun,
				Yes:     m.Yes,
				Tty:     term.IsTerminal(int(os.Stdin.Fd())),
				Stdout:  cmd.OutOrStdout(),
				Stderr:  cmd.ErrOrStderr(),
				Preview: map[string]any{"method": "DELETE", "url": path + "?clusterARN=" + clusterARN},
				ArgsLog: "clusterARN=" + clusterARN,
				Exec: func(ctx context.Context) (any, safety.ExecMeta, error) {
					resp, err := cli.Do(ctx, http.MethodDelete, path, query, nil)
					if err != nil {
						return nil, safety.ExecMeta{}, err
					}
					defer func() { _ = resp.Body.Close() }()
					if resp.StatusCode >= 400 {
						b, _ := io.ReadAll(resp.Body)
						return nil, safety.ExecMeta{}, &clierr.Error{
							Code:      codeForStatus(resp.StatusCode),
							Msg:       extractAPIMessage(b, resp.StatusCode),
							RequestID: resp.Header.Get("x-request-id"),
						}
					}
					return map[string]any{"disconnected": clusterARN}, safety.ExecMeta{URL: "DELETE " + path, Status: resp.StatusCode}, nil
				},
			})
		},
	}
	return c
}
