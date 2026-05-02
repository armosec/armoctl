package incidents

import (
	"context"
	"encoding/json"
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

func ResolveCmd(clientFor cliclient.ClientFor) *cobra.Command {
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
				Tty:     term.IsTerminal(int(os.Stdin.Fd())),
				Stdout:  cmd.OutOrStdout(),
				Stderr:  cmd.ErrOrStderr(),
				Preview: map[string]any{"method": "POST", "url": path, "body": body},
				ArgsLog: "guid=" + args[0],
				Exec: func(ctx context.Context) (any, safety.ExecMeta, error) {
					resp, err := cli.Do(ctx, http.MethodPost, path, nil, body)
					if err != nil {
						return nil, safety.ExecMeta{}, err
					}
					defer func() { _ = resp.Body.Close() }()
					raw, _ := io.ReadAll(resp.Body)
					if resp.StatusCode >= 400 {
						return nil, safety.ExecMeta{}, &clierr.Error{
							Code:      clierr.CodeServer,
							Msg:       strings.TrimSpace(string(raw)),
							RequestID: resp.Header.Get("x-request-id"),
						}
					}
					var out map[string]any
					_ = json.Unmarshal(raw, &out)
					return out, safety.ExecMeta{
						URL:       "POST " + path,
						Status:    resp.StatusCode,
						RequestID: resp.Header.Get("x-request-id"),
					}, nil
				},
			})
		},
	}
	c.Flags().Bool("false-positive", false, "Mark the incident as a false positive when resolving")
	return c
}
