package integrations

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/armosec/armoctl/internal/safety"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func UnlinkCmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{
		Use:   "unlink [guid]",
		Short: "Unlink an integration by GUID",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "unlink requires a GUID"}
			}
			guid := args[0]
			cli := clientFor(cmd)
			m := cliflags.ReadMutation(cmd)
			path := "/integrations/link/" + guid

			return safety.Wrap(cmd.Context(), safety.Args{
				Command: "integrations.unlink",
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
							Msg:       strings.TrimSpace(string(b)),
							RequestID: resp.Header.Get("x-request-id"),
						}
					}
					return map[string]any{"unlinked": guid}, safety.ExecMeta{URL: "DELETE " + path, Status: resp.StatusCode}, nil
				},
			})
		},
	}
	return c
}
