package operator

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

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

// ClusterInfo contains parsed information from an ECS cluster ARN.
type ClusterInfo struct {
	ARN         string
	Region      string
	AccountID   string
	ClusterName string
}

// parseClusterARN extracts region and cluster name from an ECS cluster ARN.
// ARN format: arn:aws:ecs:<region>:<account>:cluster/<cluster-name>
func parseClusterARN(clusterARN string) (*ClusterInfo, error) {
	parsed, err := arn.Parse(clusterARN)
	if err != nil {
		return nil, fmt.Errorf("invalid ARN: %w", err)
	}

	if parsed.Service != "ecs" {
		return nil, fmt.Errorf("expected ECS ARN, got service %q", parsed.Service)
	}

	// Resource format: "cluster/<cluster-name>"
	if !strings.HasPrefix(parsed.Resource, "cluster/") {
		return nil, fmt.Errorf("expected cluster ARN, got resource %q", parsed.Resource)
	}

	clusterName := strings.TrimPrefix(parsed.Resource, "cluster/")
	if clusterName == "" {
		return nil, fmt.Errorf("cluster name is empty in ARN")
	}

	return &ClusterInfo{
		ARN:         clusterARN,
		Region:      parsed.Region,
		AccountID:   parsed.AccountID,
		ClusterName: clusterName,
	}, nil
}

// defaultStackName returns the default CloudFormation stack name for a cluster.
func defaultStackName(clusterName string) string {
	return StackNamePrefix + clusterName
}

// defaultOperatorImage returns the default operator image for a region.
// Uses the cached version info to get the latest tag.
func defaultOperatorImage(region string) string {
	return version.GetOperatorImage(region)
}

// requireAuth returns an error if credentials are missing.
func requireAuth() error {
	if viper.GetString("customer-guid") == "" || viper.GetString("access-key") == "" {
		return fmt.Errorf(`authentication required. To get your credentials:
  1. Log in to https://%s
  2. Go to Settings > Access Keys
  3. Copy your Customer GUID and Access Key

Then either:
  - Pass as flags: armoctl --customer-guid <GUID> --access-key <KEY> ...
  - Set env vars: ARMO_CUSTOMER_GUID and ARMO_ACCESS_KEY
  - Save to config: ~/.armoctl/config.yaml`, viper.GetString("api-url"))
	}
	return nil
}
