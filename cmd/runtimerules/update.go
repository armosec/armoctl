package runtimerules

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
)

// UpdateCmd builds `armoctl runtime-rules update`.
func UpdateCmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{
		Use:   "update",
		Short: "Update a runtime rule",
		RunE: func(cmd *cobra.Command, args []string) error {
			guid, _ := cmd.Flags().GetString("guid")
			name, _ := cmd.Flags().GetString("name")
			description, _ := cmd.Flags().GetString("description")
			ruleStr, _ := cmd.Flags().GetString("rule")
			ruleFile, _ := cmd.Flags().GetString("rule-file")
			policyTypes, _ := cmd.Flags().GetStringSlice("policy-types")

			if guid == "" {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "update requires --guid"}
			}

			body := map[string]any{
				"guid": guid,
			}
			if name != "" {
				body["name"] = name
			}
			if description != "" {
				body["description"] = description
			}
			if ruleFile != "" || ruleStr != "" {
				var ruleObj map[string]any
				if ruleFile != "" {
					data, err := os.ReadFile(ruleFile)
					if err != nil {
						return &clierr.Error{Code: clierr.CodeBadInput, Msg: "failed to read --rule-file: " + err.Error()}
					}
					if err := json.Unmarshal(data, &ruleObj); err != nil {
						return &clierr.Error{Code: clierr.CodeBadInput, Msg: "failed to parse --rule-file as JSON: " + err.Error()}
					}
				} else {
					if err := json.Unmarshal([]byte(ruleStr), &ruleObj); err != nil {
						return &clierr.Error{Code: clierr.CodeBadInput, Msg: "failed to parse --rule as JSON: " + err.Error()}
					}
				}
				body["rule"] = ruleObj
			}
			if len(policyTypes) > 0 {
				body["policyTypes"] = policyTypes
			}

			const path = "/runtime/rules"
			cli := clientFor(cmd)
			m := cliflags.ReadMutation(cmd)

			return safety.Wrap(cmd.Context(), safety.Args{
				Command: "runtimerules.update",
				DryRun:  m.DryRun,
				Yes:     m.Yes,
				Tty:     false,
				Stdout:  cmd.OutOrStdout(),
				Stderr:  cmd.ErrOrStderr(),
				Preview: map[string]any{"method": "PUT", "url": path, "body": body},
				ArgsLog: "guid=" + guid,
				Exec: func(ctx context.Context) (any, safety.ExecMeta, error) {
					resp, err := cli.Do(ctx, http.MethodPut, path, nil, body)
					if err != nil {
						return nil, safety.ExecMeta{}, err
					}
					defer resp.Body.Close()
					if resp.StatusCode >= 400 {
						b, _ := io.ReadAll(resp.Body)
						return nil, safety.ExecMeta{}, &clierr.Error{Code: clierr.CodeServer, Msg: strings.TrimSpace(string(b))}
					}
					return map[string]any{"status": resp.StatusCode}, safety.ExecMeta{URL: "PUT " + path, Status: resp.StatusCode}, nil
				},
			})
		},
	}
	c.Flags().String("guid", "", "Rule GUID (required)")
	c.Flags().String("name", "", "Rule name")
	c.Flags().String("description", "", "Rule description")
	c.Flags().String("rule", "", "Rule expression as JSON string (or use --rule-file)")
	c.Flags().String("rule-file", "", "Path to JSON file containing the rule")
	c.Flags().StringSlice("policy-types", nil, "Policy types (e.g. ADR, CDR)")
	return c
}
