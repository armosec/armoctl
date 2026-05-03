package cloudaccounts

import (
	"net/url"

	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/armosec/armoctl/internal/output"
	"github.com/spf13/cobra"
)

func ECSConnectCmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{
		Use:   "connect [cluster-arn]",
		Short: "Get the CloudFormation install link for an ECS cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "connect requires the cluster ARN"}
			}
			clusterARN := args[0]
			cli := clientFor(cmd)
			query := url.Values{"clusterARN": {clusterARN}}

			var resp map[string]any
			if err := cli.GetJSON(cmd.Context(), "/accounts/ecs", query, &resp); err != nil {
				return err
			}

			return output.Render(cmd.OutOrStdout(), output.Get{Object: resp}, cliflags.OutputOptions(cmd, nil))
		},
	}
	return c
}
