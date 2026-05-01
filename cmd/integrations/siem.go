package integrations

import (
	"context"
	"encoding/json"
	"os"

	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/armosec/armoctl/internal/safety"
	"github.com/spf13/cobra"
)

func SiemCreateCmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{
		Use:   "create [provider]",
		Short: "Create SIEM integration for the given provider",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "create requires a provider name"}
			}
			provider := args[0]
			configFile, _ := cmd.Flags().GetString("config-file")

			if configFile == "" {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "create requires --config-file"}
			}

			b, err := os.ReadFile(configFile)
			if err != nil {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "failed to read config file: " + err.Error()}
			}

			var body any
			if err := json.Unmarshal(b, &body); err != nil {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "config file is not valid JSON: " + err.Error()}
			}

			cli := clientFor(cmd)
			m := cliflags.ReadMutation(cmd)
			path := "/siem/" + provider

			return safety.Wrap(cmd.Context(), safety.Args{
				Command: "integrations.siem.create",
				DryRun:  m.DryRun,
				Yes:     m.Yes,
				Tty:     false,
				Stdout:  cmd.OutOrStdout(),
				Stderr:  cmd.ErrOrStderr(),
				Preview: map[string]any{"method": "POST", "url": path, "body": body},
				ArgsLog: "provider=" + provider,
				Exec: func(ctx context.Context) (any, safety.ExecMeta, error) {
					var result any
					if err := cli.PostJSON(ctx, path, nil, body, &result); err != nil {
						return nil, safety.ExecMeta{}, err
					}
					return result, safety.ExecMeta{URL: "POST " + path, Status: 200}, nil
				},
			})
		},
	}
	c.Flags().String("config-file", "", "JSON file with provider-specific config (required)")
	return c
}
