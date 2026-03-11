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
	upgradeCmd.Flags().StringP("cluster", "c", "", "ECS cluster ARN")
	upgradeCmd.Flags().String("stack-name", "", "CloudFormation stack name (default: armo-operator-{cluster-name})")
	upgradeCmd.Flags().StringP("region", "r", "", "AWS region (required if using --stack-name without --cluster)")

	OperatorCmd.AddCommand(upgradeCmd)
}

func runUpgrade(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	clusterARN, _ := cmd.Flags().GetString("cluster")
	stackName, _ := cmd.Flags().GetString("stack-name")
	region, _ := cmd.Flags().GetString("region")

	var clusterName string

	if clusterARN != "" {
		cluster, err := parseClusterARN(clusterARN)
		if err != nil {
			return fmt.Errorf("invalid cluster ARN: %w", err)
		}
		region = cluster.Region
		clusterName = cluster.ClusterName
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

	// Get the service ARN from the stack
	output, err := DescribeStack(ctx, region, stackName)
	if err != nil {
		return fmt.Errorf("describing stack: %w", err)
	}

	if output.EcsOperatorServiceArn == "" {
		return fmt.Errorf("operator service not found in stack %q", stackName)
	}

	if clusterName == "" {
		return fmt.Errorf("--cluster is required for upgrade")
	}

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
