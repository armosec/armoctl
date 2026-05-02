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
	"golang.org/x/term"
)

func AlertChannelsCreateCmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{
		Use:   "create [guid]",
		Short: "Create an alert channel",
		RunE: func(cmd *cobra.Command, args []string) error {
			guid := ""
			if len(args) > 0 {
				guid = args[0]
			} else {
				var err error
				guid, err = cmd.Flags().GetString("guid")
				if err != nil || guid == "" {
					return &clierr.Error{Code: clierr.CodeBadInput, Msg: "create requires a GUID (positional or --guid)"}
				}
			}

			channelType, _ := cmd.Flags().GetString("type")
			configFile, _ := cmd.Flags().GetString("config-file")

			if channelType == "" {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "create requires --type"}
			}

			configData := map[string]any{}
			if configFile != "" {
				b, err := os.ReadFile(configFile)
				if err != nil {
					return &clierr.Error{Code: clierr.CodeBadInput, Msg: "failed to read config file: " + err.Error()}
				}
				if err := json.Unmarshal(b, &configData); err != nil {
					return &clierr.Error{Code: clierr.CodeBadInput, Msg: "config file is not valid JSON: " + err.Error()}
				}
			}

			body := map[string]any{"type": channelType}
			for k, v := range configData {
				body[k] = v
			}

			cli := clientFor(cmd)
			m := cliflags.ReadMutation(cmd)
			path := "/notifications/alertChannel/" + guid

			return safety.Wrap(cmd.Context(), safety.Args{
				Command: "integrations.alert-channels.create",
				DryRun:  m.DryRun,
				Yes:     m.Yes,
				Tty:     term.IsTerminal(int(os.Stdin.Fd())),
				Stdout:  cmd.OutOrStdout(),
				Stderr:  cmd.ErrOrStderr(),
				Preview: map[string]any{"method": "POST", "url": path, "body": body},
				ArgsLog: "guid=" + guid + " type=" + channelType,
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
	c.Flags().String("guid", "", "Channel GUID (optional if provided as positional)")
	c.Flags().String("type", "", "Channel type (slack|email|webhook) (required)")
	c.Flags().String("config-file", "", "JSON file with channel-specific config (optional)")
	return c
}
