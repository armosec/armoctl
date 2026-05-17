package agent

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/armosec/armoctl/ecs/clusterarn"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall the ARMO ECS Agent from an ECS cluster",
	Long: `Remove the ARMO ECS Agent from an ECS cluster by deleting the CloudFormation stack.

This will delete the agent daemon service and all associated IAM roles.

Example:
  armoctl ecs agent uninstall -c arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster

  # Skip confirmation prompt
  armoctl ecs agent uninstall -c arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster -y

  # Uninstall with explicit stack name and region
  armoctl ecs agent uninstall --stack-name armo-agent-my-cluster --region us-east-1 -y`,
	RunE: runUninstall,
}

func init() {
	uninstallCmd.Flags().StringP("cluster", "c", "", "ECS cluster ARN")
	uninstallCmd.Flags().String("stack-name", "", "CloudFormation stack name (default: armo-agent-{cluster-name})")
	uninstallCmd.Flags().StringP("region", "r", "", "AWS region (required if using --stack-name without --cluster)")
	uninstallCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")

	AgentCmd.AddCommand(uninstallCmd)
}

func runUninstall(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	clusterARN, _ := cmd.Flags().GetString("cluster")
	stackName, _ := cmd.Flags().GetString("stack-name")
	region, _ := cmd.Flags().GetString("region")
	skipConfirm, _ := cmd.Flags().GetBool("yes")

	if clusterARN != "" {
		cluster, err := clusterarn.Parse(clusterARN)
		if err != nil {
			return fmt.Errorf("invalid cluster ARN: %w", err)
		}
		region = cluster.Region
		if stackName == "" {
			stackName = defaultStackName(cluster.ClusterName)
		}
	} else if stackName != "" {
		if region == "" {
			return fmt.Errorf("--region is required when using --stack-name without --cluster")
		}
	} else {
		return fmt.Errorf("either --cluster or --stack-name is required")
	}

	if !skipConfirm {
		cmd.PrintErrf("This will delete the CloudFormation stack %q and all its resources.\n", stackName)
		cmd.PrintErrf("Delete stack %q? [y/N] ", stackName)

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading response: %w", err)
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			cmd.PrintErrln("Aborted.")
			return nil
		}
	}

	cmd.PrintErrf("Deleting stack %q...\n", stackName)

	if err := deleteStack(ctx, region, stackName); err != nil {
		return err
	}

	lastStatus := ""
	if err := waitForStackDelete(ctx, region, stackName, func(status string) {
		if status != lastStatus {
			cmd.PrintErrf("  Status: %s\n", status)
			lastStatus = status
		}
	}); err != nil {
		return err
	}

	cmd.PrintErrln()
	cmd.PrintErrln("ARMO ECS Agent uninstalled successfully.")
	return nil
}
