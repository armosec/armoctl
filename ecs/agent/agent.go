package agent

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/spf13/cobra"
)

// AgentCmd is the parent command for ECS agent operations.
var AgentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Manage the ARMO ECS Agent",
}

const (
	StackNamePrefix  = "armo-agent-"
	TemplateURL      = "https://package-distribution.armosec.io/ecs-agent/cloudformation.yaml"
)

type clusterInfo struct {
	Region      string
	ClusterName string
}

func parseClusterARN(clusterARN string) (*clusterInfo, error) {
	parsed, err := arn.Parse(clusterARN)
	if err != nil {
		return nil, fmt.Errorf("invalid ARN: %w", err)
	}
	if parsed.Service != "ecs" {
		return nil, fmt.Errorf("expected ECS ARN, got service %q", parsed.Service)
	}
	if !strings.HasPrefix(parsed.Resource, "cluster/") {
		return nil, fmt.Errorf("expected cluster ARN, got resource %q", parsed.Resource)
	}
	clusterName := strings.TrimPrefix(parsed.Resource, "cluster/")
	if clusterName == "" {
		return nil, fmt.Errorf("cluster name is empty in ARN")
	}
	return &clusterInfo{Region: parsed.Region, ClusterName: clusterName}, nil
}
