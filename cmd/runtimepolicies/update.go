package runtimepolicies

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

// UpdateCmd builds `armoctl runtime-policies update [guid]`.
func UpdateCmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{
		Use:   "update [guid]",
		Short: "Update a runtime policy",
		RunE: func(cmd *cobra.Command, args []string) error {
			guid := ""
			if len(args) > 0 {
				guid = args[0]
			}
			name, _ := cmd.Flags().GetString("name")
			description, _ := cmd.Flags().GetString("description")
			enabledFlag, _ := cmd.Flags().GetString("enabled")
			policyFile, _ := cmd.Flags().GetString("policy-file")

			if guid == "" {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "update requires a policy GUID"}
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
			if enabledFlag != "" {
				enabled := enabledFlag == "true"
				body["enabled"] = enabled
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
					if k != "guid" && k != "name" && k != "description" && k != "enabled" {
						body[k] = v
					}
				}
			}

			path := "/runtime/policies/" + guid
			cli := clientFor(cmd)
			m := cliflags.ReadMutation(cmd)

			return safety.Wrap(cmd.Context(), safety.Args{
				Command: "runtimepolicies.update",
				DryRun:  m.DryRun,
				Yes:     m.Yes,
				Tty:     term.IsTerminal(int(os.Stdin.Fd())),
				Stdout:  cmd.OutOrStdout(),
				Stderr:  cmd.ErrOrStderr(),
				Preview: map[string]any{"method": "PUT", "url": path, "body": body},
				ArgsLog: "guid=" + guid,
				Exec: func(ctx context.Context) (any, safety.ExecMeta, error) {
					resp, err := cli.Do(ctx, http.MethodPut, path, nil, body)
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
					return map[string]any{"status": resp.StatusCode}, safety.ExecMeta{URL: "PUT " + path, Status: resp.StatusCode}, nil
				},
			})
		},
	}
	c.Flags().String("name", "", "Policy name")
	c.Flags().String("description", "", "Policy description")
	c.Flags().String("enabled", "", "Enable/disable the policy (true|false)")
	c.Flags().String("policy-file", "", "Path to JSON file containing policy updates")
	return c
}
