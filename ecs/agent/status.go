package agent

import (
	"errors"
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	cftypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/spf13/cobra"

	"github.com/armosec/armoctl/ecs/clusterarn"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check the status of the ARMO ECS Agent deployment",
	Long: `Check the status of the ARMO ECS Agent CloudFormation stack.

Shows the stack status, creation time, and any stack outputs.

Example:
  armoctl ecs agent status -c arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster

  # Check status with explicit stack name and region
  armoctl ecs agent status --stack-name armo-agent-my-cluster --region us-east-1`,
	RunE: runStatus,
}

func init() {
	statusCmd.Flags().StringP("cluster", "c", "", "ECS cluster ARN")
	statusCmd.Flags().String("stack-name", "", "CloudFormation stack name (default: armo-agent-{cluster-name})")
	statusCmd.Flags().StringP("region", "r", "", "AWS region (required if using --stack-name without --cluster)")

	AgentCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	clusterARN, _ := cmd.Flags().GetString("cluster")
	stackName, _ := cmd.Flags().GetString("stack-name")
	region, _ := cmd.Flags().GetString("region")

	var clusterName string

	if clusterARN != "" {
		cluster, err := clusterarn.Parse(clusterARN)
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

	output, err := describeStackByName(ctx, region, stackName)
	if err != nil {
		var notFound *cftypes.StackNotFoundException
		if errors.As(err, &notFound) {
			if clusterName != "" {
				cmd.PrintErrf("Agent not installed in cluster %q\n", clusterName)
			} else {
				cmd.PrintErrf("Stack %q not found\n", stackName)
			}
			return nil
		}
		return err
	}

	cmd.Printf("Stack:    %s\n", output.StackName)
	cmd.Printf("Status:   %s\n", colorStatus(output.Status))
	if output.StatusReason != "" {
		cmd.Printf("Reason:   %s\n", output.StatusReason)
	}
	cmd.Printf("Created:  %s\n", output.CreationTime.Format("2006-01-02 15:04:05 MST"))
	if output.LastUpdatedTime != nil {
		cmd.Printf("Updated:  %s\n", output.LastUpdatedTime.Format("2006-01-02 15:04:05 MST"))
	}

	if len(output.Outputs) > 0 {
		cmd.Println()
		for k, v := range output.Outputs {
			cmd.Printf("%s: %s\n", k, v)
		}
	}

	if isFailedStatus(output.Status) {
		events, err := getFailedEvents(ctx, region, stackName)
		if err == nil && len(events) > 0 {
			cmd.Println()
			cmd.Printf("%s\n", redStyle.Render("Failed Resources:"))
			for _, e := range events {
				cmd.Printf("  - %s (%s)\n", e.LogicalResourceID, e.ResourceType)
				cmd.Printf("    %s\n", e.Reason)
			}
		}

		logGroup := DefaultLogGroup
		logs, err := getRecentLogs(ctx, region, logGroup, 50)
		if err == nil && len(logs) > 0 {
			cmd.Println()
			cmd.Printf("%s\n", redStyle.Render("Recent Logs ("+logGroup+"):"))
			for _, l := range logs {
				cmd.Printf("  %s  %s\n",
					yellowStyle.Render(l.Timestamp.Format("15:04:05")),
					strings.TrimSpace(l.Message),
				)
			}
		}
	}

	return nil
}

func isFailedStatus(status string) bool {
	return strings.Contains(status, "ROLLBACK") || strings.Contains(status, "FAILED")
}

var (
	greenStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
	redStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
	yellowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00"))
)

func colorStatus(status string) string {
	switch {
	case strings.HasSuffix(status, "_COMPLETE") && !strings.Contains(status, "ROLLBACK"):
		return greenStyle.Render(status)
	case strings.Contains(status, "ROLLBACK") || strings.Contains(status, "FAILED"):
		return redStyle.Render(status)
	case strings.Contains(status, "IN_PROGRESS"):
		return yellowStyle.Render(status)
	default:
		return status
	}
}
