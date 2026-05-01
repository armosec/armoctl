package runtimerules

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

// CreateCmd builds `armoctl runtime-rules create`.
func CreateCmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a runtime rule",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			description, _ := cmd.Flags().GetString("description")
			ruleStr, _ := cmd.Flags().GetString("rule")
			ruleFile, _ := cmd.Flags().GetString("rule-file")
			policyTypes, _ := cmd.Flags().GetStringSlice("policy-types")

			if name == "" {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "create requires --name"}
			}

			var ruleObj map[string]any
			if ruleFile != "" {
				data, err := os.ReadFile(ruleFile)
				if err != nil {
					return &clierr.Error{Code: clierr.CodeBadInput, Msg: "failed to read --rule-file: " + err.Error()}
				}
				if err := json.Unmarshal(data, &ruleObj); err != nil {
					return &clierr.Error{Code: clierr.CodeBadInput, Msg: "failed to parse --rule-file as JSON: " + err.Error()}
				}
			} else if ruleStr != "" {
				if err := json.Unmarshal([]byte(ruleStr), &ruleObj); err != nil {
					return &clierr.Error{Code: clierr.CodeBadInput, Msg: "failed to parse --rule as JSON: " + err.Error()}
				}
			} else {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "create requires --rule or --rule-file"}
			}

			body := map[string]any{
				"name":        name,
				"description": description,
				"type":        "Custom",
				"rule":        ruleObj,
			}
			if len(policyTypes) > 0 {
				body["policyTypes"] = policyTypes
			}

			const path = "/runtime/rules"
			cli := clientFor(cmd)
			m := cliflags.ReadMutation(cmd)

			return safety.Wrap(cmd.Context(), safety.Args{
				Command: "runtimerules.create",
				DryRun:  m.DryRun,
				Yes:     m.Yes,
				Tty:     false,
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
	c.Flags().String("name", "", "Rule name (required)")
	c.Flags().String("description", "", "Rule description")
	c.Flags().String("rule", "", "Rule expression as JSON string (or use --rule-file)")
	c.Flags().String("rule-file", "", "Path to JSON file containing the rule")
	c.Flags().StringSlice("policy-types", nil, "Policy types (e.g. ADR, CDR)")
	return c
}
