package agent

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Deploy the ARMO ECS Agent to an ECS cluster",
	Long: `Deploy the ARMO ECS Agent to an ECS cluster using CloudFormation.

The agent runs as a daemon service on every EC2 instance in the cluster and
provides runtime security monitoring.

Example:
  armoctl ecs agent install -c arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster

  # With a custom learning period
  armoctl ecs agent install -c arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster \
    --max-learning-period 6h`,
	RunE: runInstall,
}

func init() {
	installCmd.Flags().StringP("cluster", "c", "", "ECS cluster ARN (required)")
	installCmd.Flags().String("cloudwatch-logs", DefaultLogGroup, "CloudWatch log group name (empty to disable)")
	installCmd.Flags().String("stack-name", "", "CloudFormation stack name (default: armo-agent-{cluster-name})")
	installCmd.Flags().String("max-learning-period", "", "Maximum learning period (e.g. 24h, 6h, 30m; default: agent built-in 24h)")

	_ = installCmd.MarkFlagRequired("cluster")

	AgentCmd.AddCommand(installCmd)
}

func runInstall(cmd *cobra.Command, args []string) error {
	if err := requireAuth(); err != nil {
		return err
	}

	clusterARN, _ := cmd.Flags().GetString("cluster")
	cluster, err := parseClusterARN(clusterARN)
	if err != nil {
		return fmt.Errorf("invalid cluster ARN: %w", err)
	}

	stackName, _ := cmd.Flags().GetString("stack-name")
	if stackName == "" {
		stackName = defaultStackName(cluster.ClusterName)
	}

	cloudwatchLogs, _ := cmd.Flags().GetString("cloudwatch-logs")
	maxLearningPeriod, _ := cmd.Flags().GetString("max-learning-period")

	params := stackParams{
		StackName:         stackName,
		Region:            cluster.Region,
		ClusterName:       cluster.ClusterName,
		CustomerGUID:      viper.GetString("customer-guid"),
		AccessKey:         viper.GetString("access-key"),
		APIUrl:            viper.GetString("api-url"),
		CloudWatchLogs:    cloudwatchLogs,
		MaxLearningPeriod: maxLearningPeriod,
	}

	cmd.PrintErrf("Installing ARMO ECS Agent...\n")
	cmd.PrintErrf("  Cluster:  %s\n", cluster.ClusterName)
	cmd.PrintErrf("  Region:   %s\n", cluster.Region)
	cmd.PrintErrf("  Stack:    %s\n", stackName)
	if cloudwatchLogs != "" {
		cmd.PrintErrf("  Logs:     %s\n", cloudwatchLogs)
	} else {
		cmd.PrintErrln("  Logs:     (disabled)")
	}
	cmd.PrintErrln()

	ctx := cmd.Context()
	if err := createStack(ctx, params); err != nil {
		return err
	}

	cmd.PrintErrln("Stack creation initiated. Waiting for completion...")

	lastStatus := ""
	output, err := waitForStackCreate(ctx, cluster.Region, stackName, func(status string) {
		if status != lastStatus {
			cmd.PrintErrf("  Status: %s\n", status)
			lastStatus = status
		}
	})
	if err != nil {
		return err
	}

	cmd.PrintErrln()
	cmd.PrintErrln("ARMO ECS Agent installed successfully!")
	for k, v := range output.Outputs {
		cmd.PrintErrf("%s: %s\n", k, v)
	}

	return nil
}
