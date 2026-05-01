package posture

import (
	"context"

	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/armosec/armoctl/internal/safety"
	"github.com/spf13/cobra"
)

func ExceptionsCreateCmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{
		Use:   "create",
		Short: "Create a posture exception policy",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			controls, _ := cmd.Flags().GetStringArray("control")
			cluster, _ := cmd.Flags().GetString("cluster")
			namespace, _ := cmd.Flags().GetString("namespace")
			kind, _ := cmd.Flags().GetString("kind")
			workload, _ := cmd.Flags().GetString("workload")
			container, _ := cmd.Flags().GetString("container")
			reason, _ := cmd.Flags().GetString("reason")
			expires, _ := cmd.Flags().GetString("expires")

			if len(controls) == 0 {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "create requires at least one --control"}
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

			policies := make([]map[string]any, len(controls))
			for i, cid := range controls {
				policies[i] = map[string]any{"controlID": cid}
			}

			designator := map[string]any{
				"designatorType": "Attributes",
				"attributes":     attrs,
			}

			body := map[string]any{
				"name":              name,
				"policyType":        "postureExceptionPolicy",
				"actions":           []string{"alertOnly"},
				"designators":       []any{designator},
				"posturePolicies":   policies,
			}
			if reason != "" {
				body["reason"] = reason
			}
			if expires != "" {
				body["expirationDate"] = expires
			}

			const path = "/postureExceptionPolicy"
			cli := clientFor(cmd)
			m := cliflags.ReadMutation(cmd)

			return safety.Wrap(cmd.Context(), safety.Args{
				Command: "posture.exceptions.create",
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
	c.Flags().String("name", "", "Policy name")
	c.Flags().StringArray("control", nil, "Control ID to except (repeatable, required)")
	c.Flags().String("cluster", "", "Cluster name")
	c.Flags().String("namespace", "", "Namespace")
	c.Flags().String("kind", "", "Workload kind")
	c.Flags().String("workload", "", "Workload name")
	c.Flags().String("container", "", "Container name")
	c.Flags().String("reason", "", "Reason for the exception")
	c.Flags().String("expires", "", "Expiration date (RFC3339)")
	return c
}
