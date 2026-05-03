package vulns

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
		Short: "Create a vulnerability exception policy",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			cves, _ := cmd.Flags().GetStringArray("cve")
			cluster, _ := cmd.Flags().GetString("cluster")
			namespace, _ := cmd.Flags().GetString("namespace")
			kind, _ := cmd.Flags().GetString("kind")
			workload, _ := cmd.Flags().GetString("workload")
			container, _ := cmd.Flags().GetString("container")
			reason, _ := cmd.Flags().GetString("reason")
			expires, _ := cmd.Flags().GetString("expires")

			if len(cves) == 0 {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "create requires at least one --cve"}
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
			if container != "" {
				attrs["containerName"] = container
			}
			if len(attrs) == 0 {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "create requires at least one of --cluster/--namespace/--kind/--workload/--container"}
			}

			vulns := make([]map[string]string, len(cves))
			for i, cve := range cves {
				vulns[i] = map[string]string{"name": cve}
			}

			designator := map[string]any{
				"designatorType": "Attributes",
				"attributes":     attrs,
			}

			body := map[string]any{
				"name":            name,
				"policyType":      "vulnerabilityExceptionPolicy",
				"actions":         []string{"ignore"},
				"designators":     []any{designator},
				"vulnerabilities": vulns,
			}
			if reason != "" {
				body["reason"] = reason
			}
			if expires != "" {
				body["expirationDate"] = expires
			}

			const path = "/vulnerabilityExceptionPolicy"
			cli := clientFor(cmd)
			m := cliflags.ReadMutation(cmd)

			return safety.Wrap(cmd.Context(), safety.Args{
				Command: "vulns.exceptions.create",
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
	c.Flags().String("name", "", "Policy name")
	c.Flags().StringArray("cve", nil, "CVE to except (repeatable, required)")
	c.Flags().String("cluster", "", "Cluster name")
	c.Flags().String("namespace", "", "Namespace")
	c.Flags().String("kind", "", "Workload kind")
	c.Flags().String("workload", "", "Workload name")
	c.Flags().String("container", "", "Container name")
	c.Flags().String("reason", "", "Reason for the exception")
	c.Flags().String("expires", "", "Expiration date (RFC3339)")
	return c
}
