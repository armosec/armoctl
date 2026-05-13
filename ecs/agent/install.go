package agent

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Print the AWS CLI command to deploy the ARMO ECS Agent",
	Long: `Print the AWS CloudFormation deploy command for the ARMO ECS Agent.

The command is pre-filled with your credentials and cluster details.
Copy and run it to deploy the agent daemon to every EC2 instance in the cluster.

Example:
  armoctl ecs agent install -c arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster

  # With a custom learning period
  armoctl ecs agent install -c arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster \
    --max-learning-period 6h`,
	RunE: runInstall,
}

func init() {
	installCmd.Flags().StringP("cluster", "c", "", "ECS cluster ARN (required)")
	installCmd.Flags().String("cloudwatch-logs", "/armo/ecs-agent", "CloudWatch log group name (empty to disable)")
	installCmd.Flags().String("stack-name", "", "CloudFormation stack name (default: armo-agent-{cluster-name})")
	installCmd.Flags().String("max-learning-period", "", "Maximum learning period (e.g. 24h, 6h, 30m; default: agent built-in 24h)")

	_ = installCmd.MarkFlagRequired("cluster")

	AgentCmd.AddCommand(installCmd)
}

func runInstall(cmd *cobra.Command, args []string) error {
	clusterARN, _ := cmd.Flags().GetString("cluster")
	cluster, err := parseClusterARN(clusterARN)
	if err != nil {
		return fmt.Errorf("invalid cluster ARN: %w", err)
	}

	stackName, _ := cmd.Flags().GetString("stack-name")
	if stackName == "" {
		stackName = StackNamePrefix + cluster.ClusterName
	}

	cloudwatchLogs, _ := cmd.Flags().GetString("cloudwatch-logs")
	maxLearningPeriod, _ := cmd.Flags().GetString("max-learning-period")

	customerGUID := viper.GetString("customer-guid")
	accessKey := viper.GetString("access-key")
	apiURL := viper.GetString("api-url")

	overrides := []string{
		fmt.Sprintf("Region=%s", cluster.Region),
		fmt.Sprintf("ClusterName=%s", cluster.ClusterName),
		fmt.Sprintf("CustomerGUID=%s", customerGUID),
		fmt.Sprintf("AccessKey=%s", accessKey),
		fmt.Sprintf("ApiUrl=%s", apiURL),
		fmt.Sprintf("CloudWatchLogsGroupName=%s", cloudwatchLogs),
	}
	if maxLearningPeriod != "" {
		overrides = append(overrides, fmt.Sprintf("MaxLearningPeriod=%s", maxLearningPeriod))
	}

	cmd.Printf("Run the following command to deploy the ARMO ECS Agent:\n\n")
	cmd.Printf("  curl -Lo cloudformation.yaml %s\n\n", TemplateURL)
	cmd.Printf("  aws cloudformation deploy \\\n")
	cmd.Printf("    --template-file cloudformation.yaml \\\n")
	cmd.Printf("    --stack-name %s \\\n", stackName)
	cmd.Printf("    --region %s \\\n", cluster.Region)
	cmd.Printf("    --capabilities CAPABILITY_NAMED_IAM \\\n")
	cmd.Printf("    --parameter-overrides \\\n")
	for i, o := range overrides {
		if i < len(overrides)-1 {
			cmd.Printf("      %s \\\n", o)
		} else {
			cmd.Printf("      %s\n", o)
		}
	}

	cmd.Println()
	cmd.Printf("To uninstall:\n\n")
	cmd.Printf("  aws cloudformation delete-stack --stack-name %s --region %s\n", stackName, cluster.Region)
	cmd.Println()
	cmd.Printf("To check status:\n\n")
	cmd.Printf("  aws cloudformation describe-stacks --stack-name %s --region %s\n", stackName, cluster.Region)

	return nil
}
