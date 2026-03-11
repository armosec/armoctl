package operator

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall the ARMO ECS Operator from an ECS cluster",
	Long: `Remove the ARMO ECS Operator from an ECS cluster by deleting the CloudFormation stack.

This will delete:
  - The ECS Operator service
  - The ECS Operator task definition
  - The IAM roles created for the operator

Example:
  armoctl ecs operator uninstall -c arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster

  # Skip confirmation prompt
  armoctl ecs operator uninstall -c arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster -y

  # Uninstall with explicit stack name and region
  armoctl ecs operator uninstall --stack-name armo-operator-my-cluster --region us-east-1 -y`,
	RunE: runUninstall,
}

func init() {
	uninstallCmd.Flags().StringP("cluster", "c", "", "ECS cluster ARN")
	uninstallCmd.Flags().String("stack-name", "", "CloudFormation stack name (default: armo-operator-{cluster-name})")
	uninstallCmd.Flags().StringP("region", "r", "", "AWS region (required if using --stack-name without --cluster)")
	uninstallCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")

	OperatorCmd.AddCommand(uninstallCmd)
}

func runUninstall(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Get flags
	clusterARN, _ := cmd.Flags().GetString("cluster")
	stackName, _ := cmd.Flags().GetString("stack-name")
	region, _ := cmd.Flags().GetString("region")
	skipConfirm, _ := cmd.Flags().GetBool("yes")

	// Determine region and stack name
	if clusterARN != "" {
		cluster, err := parseClusterARN(clusterARN)
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

	// Confirm deletion
	if !skipConfirm {
		fmt.Fprintf(os.Stderr, "This will delete the CloudFormation stack %q and all its resources.\n", stackName)
		fmt.Fprintf(os.Stderr, "Delete stack %q? [y/N] ", stackName)

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("reading response: %w", err)
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Fprintln(os.Stderr, "Aborted.")
			return nil
		}
	}

	// Delete the stack
	fmt.Fprintf(os.Stderr, "Deleting stack %q...\n", stackName)

	if err := DeleteStack(ctx, region, stackName); err != nil {
		return err
	}

	// Wait for deletion with progress updates
	lastStatus := ""
	err := WaitForStackDelete(ctx, region, stackName, func(status string) {
		if status != lastStatus {
			fmt.Fprintf(os.Stderr, "  Status: %s\n", status)
			lastStatus = status
		}
	})
	if err != nil {
		return err
	}

	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "ARMO ECS Operator uninstalled successfully.\n")

	return nil
}
