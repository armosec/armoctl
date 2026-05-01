package runtimerules

import (
	"encoding/json"
	"os"

	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/armosec/armoctl/internal/output"
	"github.com/spf13/cobra"
)

// EvaluateCmd builds `armoctl runtime-rules evaluate`.
func EvaluateCmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{
		Use:   "evaluate",
		Short: "Evaluate a runtime rule against input data",
		RunE: func(cmd *cobra.Command, args []string) error {
			ruleStr, _ := cmd.Flags().GetString("rule")
			ruleFile, _ := cmd.Flags().GetString("rule-file")
			inputStr, _ := cmd.Flags().GetString("input")
			inputFile, _ := cmd.Flags().GetString("input-file")

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
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "evaluate requires --rule or --rule-file"}
			}

			var inputObj map[string]any
			if inputFile != "" {
				data, err := os.ReadFile(inputFile)
				if err != nil {
					return &clierr.Error{Code: clierr.CodeBadInput, Msg: "failed to read --input-file: " + err.Error()}
				}
				if err := json.Unmarshal(data, &inputObj); err != nil {
					return &clierr.Error{Code: clierr.CodeBadInput, Msg: "failed to parse --input-file as JSON: " + err.Error()}
				}
			} else if inputStr != "" {
				if err := json.Unmarshal([]byte(inputStr), &inputObj); err != nil {
					return &clierr.Error{Code: clierr.CodeBadInput, Msg: "failed to parse --input as JSON: " + err.Error()}
				}
			} else {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "evaluate requires --input or --input-file"}
			}

			body := map[string]any{
				"rule":  ruleObj,
				"input": inputObj,
			}

			cli := clientFor(cmd)
			const path = "/runtime/rules/evaluate"
			var resp map[string]any
			if err := cli.PostJSON(cmd.Context(), path, nil, body, &resp); err != nil {
				return err
			}
			return output.Render(cmd.OutOrStdout(), output.Get{Object: resp}, cliflags.OutputOptions(cmd, nil))
		},
	}
	c.Flags().String("rule", "", "Rule expression as JSON string (or use --rule-file)")
	c.Flags().String("rule-file", "", "Path to JSON file containing the rule")
	c.Flags().String("input", "", "Input data as JSON string (or use --input-file)")
	c.Flags().String("input-file", "", "Path to JSON file containing the input data")
	return c
}
