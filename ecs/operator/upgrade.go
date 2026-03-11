package operator

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade the ARMO ECS Operator to the latest image",
	Long: `Force a new deployment of the ARMO ECS Operator service, pulling the latest container image.

This triggers an ECS rolling deployment that replaces the running task with a new one
using the latest image.

Example:
  armoctl ecs operator upgrade -c arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster`,
	RunE: runUpgrade,
}

func init() {
	upgradeCmd.Flags().StringP("cluster", "c", "", "ECS cluster ARN (required)")

	_ = upgradeCmd.MarkFlagRequired("cluster")

	OperatorCmd.AddCommand(upgradeCmd)
}

func runUpgrade(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	clusterARN, _ := cmd.Flags().GetString("cluster")

	cluster, err := parseClusterARN(clusterARN)
	if err != nil {
		return fmt.Errorf("invalid cluster ARN: %w", err)
	}

	stackName := defaultStackName(cluster.ClusterName)

	// Get the service ARN from the stack
	output, err := DescribeStack(ctx, cluster.Region, stackName)
	if err != nil {
		return fmt.Errorf("describing stack: %w", err)
	}

	if output.EcsOperatorServiceArn == "" {
		return fmt.Errorf("operator service not found in stack %q", stackName)
	}

	clusterName := cluster.ClusterName
	region := cluster.Region

	fmt.Fprintf(os.Stderr, "Forcing new deployment for ARMO ECS Operator...\n")
	fmt.Fprintf(os.Stderr, "  Cluster: %s\n", clusterName)
	fmt.Fprintf(os.Stderr, "  Service: %s\n", output.EcsOperatorServiceArn)
	fmt.Fprintln(os.Stderr)

	if err := ForceNewDeployment(ctx, region, clusterName, "armo-ecs-operator"); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Upgrade initiated. The operator will pull the latest image and restart.\n")

	return nil
}
