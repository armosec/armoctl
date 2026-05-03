package risks

import (
	"context"
	"io"
	"net/http"
	"os"

	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/armosec/armoctl/internal/safety"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func ExceptionsDeleteCmd(clientFor cliclient.ClientFor) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <guid>",
		Short: "Delete a security-risk exception policy by GUID",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "delete requires the exception GUID"}
			}
			guid := args[0]
			cli := clientFor(cmd)
			m := cliflags.ReadMutation(cmd)
			path := "/securityrisks/exceptions/" + guid

			return safety.Wrap(cmd.Context(), safety.Args{
				Command: "risks.exceptions.delete",
				DryRun:  m.DryRun,
				Yes:     m.Yes,
				Tty:     term.IsTerminal(int(os.Stdin.Fd())),
				Stdout:  cmd.OutOrStdout(),
				Stderr:  cmd.ErrOrStderr(),
				Preview: map[string]any{"method": "DELETE", "url": path},
				ArgsLog: "guid=" + guid,
				Exec: func(ctx context.Context) (any, safety.ExecMeta, error) {
					resp, err := cli.Do(ctx, http.MethodDelete, path, nil, nil)
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
					return map[string]any{"deleted": guid}, safety.ExecMeta{URL: "DELETE " + path, Status: resp.StatusCode}, nil
				},
			})
		},
	}
}
