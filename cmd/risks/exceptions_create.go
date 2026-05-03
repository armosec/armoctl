package risks

import (
	"context"
	"os"

	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/armosec/armoctl/internal/safety"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func ExceptionsCreateCmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a security-risk exception policy (accept a risk)",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			riskID, _ := cmd.Flags().GetString("risk-id")
			reason, _ := cmd.Flags().GetString("reason")
			expires, _ := cmd.Flags().GetString("expires")
			cluster, _ := cmd.Flags().GetString("cluster")
			namespace, _ := cmd.Flags().GetString("namespace")
			kind, _ := cmd.Flags().GetString("kind")
			workload, _ := cmd.Flags().GetString("workload")

			if riskID == "" {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "create requires --risk-id"}
			}

			body := map[string]any{
				"name":       name,
				"policyType": "securityRiskExceptionPolicy",
				"policyIDs":  []string{riskID},
			}
			if reason != "" {
				body["reason"] = reason
			}
			if expires != "" {
				body["expirationDate"] = expires
			}

			attrs := map[string]string{}
			if cluster != "" {
				attrs["cluster"] = cluster
			}
			if namespace != "" {
				attrs["namespace"] = namespace
			}
			if kind != "" {
				attrs["kind"] = kind
			}
			if workload != "" {
				attrs["name"] = workload
			}
			if len(attrs) > 0 {
				body["resources"] = []any{map[string]any{
					"designatorType": "Attributes",
					"attributes":     attrs,
				}}
			}

			const path = "/securityrisks/exceptions/new"
			cli := clientFor(cmd)
			m := cliflags.ReadMutation(cmd)

			return safety.Wrap(cmd.Context(), safety.Args{
				Command: "risks.exceptions.create",
				DryRun:  m.DryRun,
				Yes:     m.Yes,
				Tty:     term.IsTerminal(int(os.Stdin.Fd())),
				Stdout:  cmd.OutOrStdout(),
				Stderr:  cmd.ErrOrStderr(),
				Preview: map[string]any{"method": "POST", "url": path, "body": body},
				ArgsLog: "riskID=" + riskID,
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
	c.Flags().String("name", "", "Policy name")
	c.Flags().String("risk-id", "", "Security risk ID to accept (required)")
	c.Flags().String("reason", "", "Reason for accepting the risk")
	c.Flags().String("expires", "", "Expiration date (RFC3339)")
	c.Flags().String("cluster", "", "Optional resource scope: cluster name")
	c.Flags().String("namespace", "", "Optional resource scope: namespace")
	c.Flags().String("kind", "", "Optional resource scope: workload kind")
	c.Flags().String("workload", "", "Optional resource scope: workload name")
	return c
}
