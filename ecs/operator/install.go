package operator

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/armosec/armoctl/cmd/cliclient"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the ARMO ECS Operator in an ECS cluster",
	Long: `Deploy the ARMO ECS Operator to an ECS cluster using CloudFormation.

The operator runs as a single replica ECS service and provides cluster visibility
for ARMO security monitoring.

Example:
  armoctl ecs operator install -c arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster

  # With custom log group
  armoctl ecs operator install -c arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster \
    --cloudwatch-logs /custom/logs

  # Disable CloudWatch logging
  armoctl ecs operator install -c arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster \
    --cloudwatch-logs ""`,
	RunE: runInstall,
}

func init() {
	installCmd.Flags().StringP("cluster", "c", "", "ECS cluster ARN (required)")
	installCmd.Flags().String("operator-image", "", "ECS Operator container image (default: ARMO ECR image for the cluster region)")
	installCmd.Flags().String("cloudwatch-logs", DefaultLogGroup, "CloudWatch log group name (empty to disable)")
	installCmd.Flags().String("stack-name", "", "CloudFormation stack name (default: armo-operator-{cluster-name})")

	_ = installCmd.MarkFlagRequired("cluster")

	OperatorCmd.AddCommand(installCmd)
}

func runInstall(cmd *cobra.Command, args []string) error {
	// Require authentication
	if err := requireAuth(); err != nil {
		return err
	}

	// Parse cluster ARN
	clusterARN, _ := cmd.Flags().GetString("cluster")
	cluster, err := parseClusterARN(clusterARN)
	if err != nil {
		return fmt.Errorf("invalid cluster ARN: %w", err)
	}

	// Register the cluster with ARMO (as the UI does): it shows up in the UI
	// and the cluster's AWS account may pull ARMO's ECS images.
	ctx := cmd.Context()
	client := cliclient.Default(viper.GetString)(cmd)
	if err := registerCluster(ctx, client, clusterARN, registerTimeout, registerInterval); err != nil {
		return err
	}
	cmd.PrintErrln("Registered cluster with ARMO and granted image pull access")

	operatorImage, err := operatorImageFor(cmd, client, cluster.Region)
	if err != nil {
		return err
	}

	stackName, _ := cmd.Flags().GetString("stack-name")
	if stackName == "" {
		stackName = defaultStackName(cluster.ClusterName)
	}

	cloudwatchLogs, _ := cmd.Flags().GetString("cloudwatch-logs")

	// Build stack parameters
	params := StackParams{
		StackName:      stackName,
		Region:         cluster.Region,
		ClusterName:    cluster.ClusterName,
		CustomerGUID:   viper.GetString("customer-guid"),
		AccessKey:      viper.GetString("access-key"),
		APIUrl:         viper.GetString("api-url"),
		OperatorImage:  operatorImage,
		CloudWatchLogs: cloudwatchLogs,
	}

	// Print installation info
	cmd.PrintErrf("Installing ARMO ECS Operator...\n")
	cmd.PrintErrf("  Cluster:    %s\n", cluster.ClusterName)
	cmd.PrintErrf("  Region:     %s\n", cluster.Region)
	cmd.PrintErrf("  Stack:      %s\n", stackName)
	cmd.PrintErrf("  Image:      %s\n", operatorImage)
	if cloudwatchLogs != "" {
		cmd.PrintErrf("  Log Group:  %s\n", cloudwatchLogs)
	} else {
		cmd.PrintErrln("  Log Group:  (disabled)")
	}
	cmd.PrintErrln()

	// Create the stack
	if err := CreateStack(ctx, params); err != nil {
		return err
	}

	cmd.PrintErrln("Stack creation initiated. Waiting for completion...")

	// Wait for stack creation with progress updates
	lastStatus := ""
	output, err := WaitForStackCreate(ctx, cluster.Region, stackName, func(status string) {
		if status != lastStatus {
			cmd.PrintErrf("  Status: %s\n", status)
			lastStatus = status
		}
	})
	if err != nil {
		return err
	}

	// Print success message
	cmd.PrintErrln()
	cmd.PrintErrln("ARMO ECS Operator installed successfully!")
	cmd.PrintErrln()
	cmd.PrintErrf("Service ARN:         %s\n", output.EcsOperatorServiceArn)
	cmd.PrintErrf("Task Definition ARN: %s\n", output.EcsOperatorTaskDefinitionArn)

	return nil
}
