package ecs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awsecs "github.com/aws/aws-sdk-go-v2/service/ecs"

	"github.com/armosec/armoctl/ecs/patcher"
)

// newECSClient creates an ECS client from the default AWS config.
func newECSClient(ctx context.Context) (*awsecs.Client, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}
	return awsecs.NewFromConfig(cfg), nil
}

// describeTaskDef fetches a task definition by ARN and converts it to
// RegisterTaskDefinitionInput via JSON round-trip. This works because the
// response type is a superset of the input type, and json.Unmarshal performs
// case-insensitive field matching.
func describeTaskDef(ctx context.Context, client *awsecs.Client, arn string) (*patcher.TaskDefinition, error) {
	out, err := client.DescribeTaskDefinition(ctx, &awsecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String(arn),
	})
	if err != nil {
		return nil, fmt.Errorf("describing task definition: %w", err)
	}

	data, err := json.Marshal(out.TaskDefinition)
	if err != nil {
		return nil, fmt.Errorf("marshaling AWS task definition: %w", err)
	}

	return patcher.UnmarshalTaskDef(data)
}

// registerTaskDef registers a task definition with AWS ECS and returns the new ARN.
func registerTaskDef(ctx context.Context, client *awsecs.Client, td *patcher.TaskDefinition) (string, error) {
	out, err := client.RegisterTaskDefinition(ctx, td)
	if err != nil {
		return "", fmt.Errorf("calling RegisterTaskDefinition: %w", err)
	}

	if out.TaskDefinition != nil && out.TaskDefinition.TaskDefinitionArn != nil {
		return *out.TaskDefinition.TaskDefinitionArn, nil
	}
	return "", nil
}
