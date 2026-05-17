package operator

import (
	"github.com/spf13/cobra"

	"github.com/armosec/armoctl/internal/config"
	"github.com/armosec/armoctl/internal/version"
)

// OperatorCmd is the parent command for ECS operator operations.
var OperatorCmd = &cobra.Command{
	Use:   "operator",
	Short: "Manage the ARMO ECS Operator",
	Long:  "Deploy, manage, and remove the ARMO ECS Operator in an ECS cluster using CloudFormation.",
}

const (
	// DefaultLogGroup is the default CloudWatch log group for the operator.
	DefaultLogGroup = "/armo/ecs-operator"

	// StackNamePrefix is the prefix for CloudFormation stack names.
	StackNamePrefix = "armo-operator-"
)

// defaultStackName returns the default CloudFormation stack name for a cluster.
func defaultStackName(clusterName string) string {
	return StackNamePrefix + clusterName
}

// defaultOperatorImage returns the default operator image for a region.
// Uses the cached version info to get the latest tag.
func defaultOperatorImage(region string) string {
	return version.GetOperatorImage(region)
}

// requireAuth checks credentials, prompting interactively if missing.
func requireAuth() error {
	return config.RequireAuth()
}
