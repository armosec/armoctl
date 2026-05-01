package runtimerules

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/armosec/armoctl/internal/safety"
	"github.com/spf13/cobra"
)

// DeleteCmd builds `armoctl runtime-rules delete [ruleGUID]`.
func DeleteCmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{
		Use:   "delete [ruleGUID]",
		Short: "Delete a runtime rule",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "delete requires a rule GUID"}
			}
			guid := args[0]
			path := "/runtime/rules/" + guid
			cli := clientFor(cmd)
			m := cliflags.ReadMutation(cmd)

			return safety.Wrap(cmd.Context(), safety.Args{
				Command: "runtimerules.delete",
				DryRun:  m.DryRun,
				Yes:     m.Yes,
				Tty:     false,
				Stdout:  cmd.OutOrStdout(),
				Stderr:  cmd.ErrOrStderr(),
				Preview: map[string]any{"method": "DELETE", "url": path},
				ArgsLog: "guid=" + guid,
				Exec: func(ctx context.Context) (any, safety.ExecMeta, error) {
					resp, err := cli.Do(ctx, http.MethodDelete, path, nil, nil)
					if err != nil {
						return nil, safety.ExecMeta{}, err
					}
					defer resp.Body.Close()
					if resp.StatusCode >= 400 {
						b, _ := io.ReadAll(resp.Body)
						return nil, safety.ExecMeta{}, &clierr.Error{Code: clierr.CodeServer, Msg: strings.TrimSpace(string(b))}
					}
					return map[string]any{"status": resp.StatusCode}, safety.ExecMeta{URL: "DELETE " + path, Status: resp.StatusCode}, nil
				},
			})
		},
	}
	return c
}
