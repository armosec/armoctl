package agent

import (
	"github.com/armosec/armoctl/internal/config"
	"github.com/spf13/cobra"
)

// AgentCmd is the parent command for ECS agent operations.
var AgentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Manage the ARMO ECS Agent",
}

const (
	StackNamePrefix = "armo-agent-"
	TemplateURL     = "https://package-distribution.armosec.io/ecs-agent/cloudformation.yaml"
	DefaultLogGroup = "/armo/ecs-agent"
)

func requireAuth() error {
	return config.RequireAuth()
}

func defaultStackName(clusterName string) string {
	return StackNamePrefix + clusterName
}

