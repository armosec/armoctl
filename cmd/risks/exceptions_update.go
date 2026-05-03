package risks

import (
	"context"
	"io"
	"net/http"
	"os"

	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/armosec/armoctl/internal/safety"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func ExceptionsUpdateCmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{
		Use:   "update",
		Short: "Update an existing security-risk exception policy",
		RunE: func(cmd *cobra.Command, args []string) error {
			guid, _ := cmd.Flags().GetString("guid")
			riskID, _ := cmd.Flags().GetString("risk-id")
			if guid == "" {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "update requires --guid"}
			}
			if riskID == "" {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "update requires --risk-id"}
			}

			// Optional fields included only when explicitly set, so users can
			// clear a field via --reason "".
			nameSet := cmd.Flags().Changed("name")
			reasonSet := cmd.Flags().Changed("reason")
			expiresSet := cmd.Flags().Changed("expires")
			name, _ := cmd.Flags().GetString("name")
			reason, _ := cmd.Flags().GetString("reason")
			expires, _ := cmd.Flags().GetString("expires")

			body := map[string]any{
				"guid":       guid,
				"policyType": "securityRiskExceptionPolicy",
				"policyIDs":  []string{riskID},
			}
			if nameSet {
				body["name"] = name
			}
			if reasonSet {
				body["reason"] = reason
			}
			if expiresSet {
				body["expirationDate"] = expires
			}

			const path = "/securityrisks/exceptions"
			cli := clientFor(cmd)
			m := cliflags.ReadMutation(cmd)

			return safety.Wrap(cmd.Context(), safety.Args{
				Command: "risks.exceptions.update",
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
							Msg:       extractAPIMessage(b, resp.StatusCode),
							RequestID: resp.Header.Get("x-request-id"),
						}
					}
					return map[string]any{"status": resp.StatusCode}, safety.ExecMeta{URL: "PUT " + path, Status: resp.StatusCode}, nil
				},
			})
		},
	}
	c.Flags().String("guid", "", "Exception policy GUID (required)")
	c.Flags().String("risk-id", "", "Security risk ID (required)")
	c.Flags().String("name", "", "Policy name")
	c.Flags().String("reason", "", "Reason")
	c.Flags().String("expires", "", "Expiration date (RFC3339)")
	return c
}
