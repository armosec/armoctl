package runtimepolicies

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

// CreateCmd builds `armoctl runtime-policies create`.
func CreateCmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a runtime policy",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			description, _ := cmd.Flags().GetString("description")
			enabled, _ := cmd.Flags().GetBool("enabled")
			policyFile, _ := cmd.Flags().GetString("policy-file")

			if name == "" {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "create requires --name"}
			}

			body := map[string]any{
				"name":        name,
				"description": description,
				"enabled":     enabled,
			}

			if policyFile != "" {
				data, err := os.ReadFile(policyFile)
				if err != nil {
					return &clierr.Error{Code: clierr.CodeBadInput, Msg: "failed to read --policy-file: " + err.Error()}
				}
				var policyData map[string]any
				if err := json.Unmarshal(data, &policyData); err != nil {
					return &clierr.Error{Code: clierr.CodeBadInput, Msg: "failed to parse --policy-file as JSON: " + err.Error()}
				}
				for k, v := range policyData {
					if k != "name" && k != "description" && k != "enabled" {
						body[k] = v
					}
				}
			}

			const path = "/runtime/policies"
			cli := clientFor(cmd)
			m := cliflags.ReadMutation(cmd)

			return safety.Wrap(cmd.Context(), safety.Args{
				Command: "runtimepolicies.create",
				DryRun:  m.DryRun,
				Yes:     m.Yes,
				Tty:     term.IsTerminal(int(os.Stdin.Fd())),
				Stdout:  cmd.OutOrStdout(),
				Stderr:  cmd.ErrOrStderr(),
				Preview: map[string]any{"method": "POST", "url": path, "body": body},
				ArgsLog: "name=" + name,
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
	c.Flags().String("name", "", "Policy name (required)")
	c.Flags().String("description", "", "Policy description")
	c.Flags().Bool("enabled", true, "Enable the policy")
	c.Flags().String("policy-file", "", "Path to JSON file containing the full policy body")
	return c
}
