// Package clusterarn parses ECS cluster ARNs.
package clusterarn

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
)

// Info is the parsed result of an ECS cluster ARN.
// ARN format: arn:aws:ecs:<region>:<account>:cluster/<cluster-name>
type Info struct {
	Region      string
	ClusterName string
}

// Parse extracts region and cluster name from an ECS cluster ARN.
func Parse(clusterARN string) (*Info, error) {
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
	return &Info{Region: parsed.Region, ClusterName: clusterName}, nil
}
