package vulns

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
		Short: "Update an existing vulnerability exception policy",
		RunE: func(cmd *cobra.Command, args []string) error {
			guid, _ := cmd.Flags().GetString("guid")
			cves, _ := cmd.Flags().GetStringArray("cve")
			cluster, _ := cmd.Flags().GetString("cluster")
			namespace, _ := cmd.Flags().GetString("namespace")
			kind, _ := cmd.Flags().GetString("kind")
			workload, _ := cmd.Flags().GetString("workload")
			container, _ := cmd.Flags().GetString("container")
			// Optional fields: include in body only if the flag was explicitly set
			// (so users can intentionally clear a field via --name "").
			nameSet := cmd.Flags().Changed("name")
			reasonSet := cmd.Flags().Changed("reason")
			expiresSet := cmd.Flags().Changed("expires")
			name, _ := cmd.Flags().GetString("name")
			reason, _ := cmd.Flags().GetString("reason")
			expires, _ := cmd.Flags().GetString("expires")

			if guid == "" {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "update requires --guid"}
			}
			if len(cves) == 0 {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "update requires at least one --cve"}
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
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "update requires at least one of --cluster/--namespace/--kind/--workload/--container"}
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
				"guid":            guid,
				"policyType":      "vulnerabilityExceptionPolicy",
				"actions":         []string{"ignore"},
				"designators":     []any{designator},
				"vulnerabilities": vulns,
			}
			// Only include optional fields when their flags were explicitly set.
			if nameSet {
				body["name"] = name
			}
			if reasonSet {
				body["reason"] = reason
			}
			if expiresSet {
				body["expirationDate"] = expires
			}

			const path = "/vulnerabilityExceptionPolicy"
			cli := clientFor(cmd)
			m := cliflags.ReadMutation(cmd)

			return safety.Wrap(cmd.Context(), safety.Args{
				Command: "vulns.exceptions.update",
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
	c.Flags().String("guid", "", "Policy GUID (required)")
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
