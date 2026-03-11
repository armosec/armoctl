package operator

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	cftypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

// StackParams contains the parameters for creating a CloudFormation stack.
type StackParams struct {
	StackName      string
	Region         string
	ClusterName    string
	CustomerGUID   string
	AccessKey      string
	APIUrl         string
	OperatorImage  string
	CloudWatchLogs string
}

// StackOutput contains the outputs from a CloudFormation stack.
type StackOutput struct {
	StackName                    string
	Status                       string
	StatusReason                 string
	CreationTime                 time.Time
	LastUpdatedTime              *time.Time
	EcsOperatorServiceArn        string
	EcsOperatorTaskDefinitionArn string
}

// newCFClient creates a CloudFormation client for the specified region.
func newCFClient(ctx context.Context, region string) (*cloudformation.Client, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}
	return cloudformation.NewFromConfig(cfg), nil
}

// CreateStack creates a new CloudFormation stack with the operator template.
func CreateStack(ctx context.Context, params StackParams) error {
	client, err := newCFClient(ctx, params.Region)
	if err != nil {
		return err
	}

	// Check if stack already exists
	exists, err := stackExists(ctx, client, params.StackName)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("stack %q already exists. Use 'armoctl ecs operator uninstall' first", params.StackName)
	}

	// Build CloudFormation parameters
	cfParams := []cftypes.Parameter{
		{ParameterKey: aws.String("Region"), ParameterValue: aws.String(params.Region)},
		{ParameterKey: aws.String("EcsClusterName"), ParameterValue: aws.String(params.ClusterName)},
		{ParameterKey: aws.String("CustomerGuid"), ParameterValue: aws.String(params.CustomerGUID)},
		{ParameterKey: aws.String("AccessKey"), ParameterValue: aws.String(params.AccessKey)},
		{ParameterKey: aws.String("ApiUrl"), ParameterValue: aws.String(params.APIUrl)},
		{ParameterKey: aws.String("EcsOperatorImage"), ParameterValue: aws.String(params.OperatorImage)},
		{ParameterKey: aws.String("CloudWatchLogsGroupName"), ParameterValue: aws.String(params.CloudWatchLogs)},
	}

	_, err = client.CreateStack(ctx, &cloudformation.CreateStackInput{
		StackName:    aws.String(params.StackName),
		TemplateBody: aws.String(CloudFormationTemplate),
		Parameters:   cfParams,
		Capabilities: []cftypes.Capability{
			cftypes.CapabilityCapabilityNamedIam,
		},
		Tags: []cftypes.Tag{
			{Key: aws.String("armo.io/managed-by"), Value: aws.String("armoctl")},
			{Key: aws.String("armo.io/cluster"), Value: aws.String(params.ClusterName)},
		},
	})
	if err != nil {
		return fmt.Errorf("creating stack: %w", err)
	}

	return nil
}

// WaitForStackCreate waits for the stack creation to complete.
// It calls the progress callback with status updates.
func WaitForStackCreate(ctx context.Context, region, stackName string, progress func(status string)) (*StackOutput, error) {
	client, err := newCFClient(ctx, region)
	if err != nil {
		return nil, err
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			output, err := describeStack(ctx, client, stackName)
			if err != nil {
				return nil, err
			}

			if progress != nil {
				progress(output.Status)
			}

			switch output.Status {
			case string(cftypes.StackStatusCreateComplete):
				return output, nil
			case string(cftypes.StackStatusCreateFailed),
				string(cftypes.StackStatusRollbackComplete),
				string(cftypes.StackStatusRollbackFailed):
				reason := output.StatusReason
				if reason == "" {
					reason = "check CloudFormation console for details"
				}
				return output, fmt.Errorf("stack creation failed: %s - %s", output.Status, reason)
			}
			// Still in progress, continue waiting
		}
	}
}

// DeleteStack deletes a CloudFormation stack.
func DeleteStack(ctx context.Context, region, stackName string) error {
	client, err := newCFClient(ctx, region)
	if err != nil {
		return err
	}

	// Check if stack exists
	exists, err := stackExists(ctx, client, stackName)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("stack %q not found", stackName)
	}

	_, err = client.DeleteStack(ctx, &cloudformation.DeleteStackInput{
		StackName: aws.String(stackName),
	})
	if err != nil {
		return fmt.Errorf("deleting stack: %w", err)
	}

	return nil
}

// WaitForStackDelete waits for the stack deletion to complete.
func WaitForStackDelete(ctx context.Context, region, stackName string, progress func(status string)) error {
	client, err := newCFClient(ctx, region)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			output, err := describeStack(ctx, client, stackName)
			if err != nil {
				// Stack not found means deletion is complete
				var notFound *cftypes.StackNotFoundException
				if errors.As(err, &notFound) {
					return nil
				}
				return err
			}

			if progress != nil {
				progress(output.Status)
			}

			switch output.Status {
			case string(cftypes.StackStatusDeleteComplete):
				return nil
			case string(cftypes.StackStatusDeleteFailed):
				reason := output.StatusReason
				if reason == "" {
					reason = "check CloudFormation console for details"
				}
				return fmt.Errorf("stack deletion failed: %s", reason)
			}
			// Still in progress, continue waiting
		}
	}
}

// DescribeStack returns information about a CloudFormation stack.
func DescribeStack(ctx context.Context, region, stackName string) (*StackOutput, error) {
	client, err := newCFClient(ctx, region)
	if err != nil {
		return nil, err
	}
	return describeStack(ctx, client, stackName)
}

// stackExists checks if a CloudFormation stack exists and is not deleted.
func stackExists(ctx context.Context, client *cloudformation.Client, stackName string) (bool, error) {
	output, err := client.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	})
	if err != nil {
		var notFound *cftypes.StackNotFoundException
		if errors.As(err, &notFound) {
			return false, nil
		}
		// Check for "does not exist" in error message (sometimes returned instead of typed error)
		if isStackNotFoundError(err) {
			return false, nil
		}
		return false, fmt.Errorf("describing stack: %w", err)
	}

	if len(output.Stacks) == 0 {
		return false, nil
	}

	// Check if stack is in a deleted state
	status := output.Stacks[0].StackStatus
	if status == cftypes.StackStatusDeleteComplete {
		return false, nil
	}

	return true, nil
}

// isStackNotFoundError checks if the error indicates a stack was not found.
func isStackNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return contains(errMsg, "does not exist") || contains(errMsg, "not found")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// describeStack returns information about a CloudFormation stack.
func describeStack(ctx context.Context, client *cloudformation.Client, stackName string) (*StackOutput, error) {
	output, err := client.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	})
	if err != nil {
		var notFound *cftypes.StackNotFoundException
		if errors.As(err, &notFound) {
			return nil, notFound
		}
		if isStackNotFoundError(err) {
			return nil, &cftypes.StackNotFoundException{Message: aws.String("stack not found")}
		}
		return nil, fmt.Errorf("describing stack: %w", err)
	}

	if len(output.Stacks) == 0 {
		return nil, &cftypes.StackNotFoundException{Message: aws.String("stack not found")}
	}

	stack := output.Stacks[0]
	result := &StackOutput{
		StackName:    aws.ToString(stack.StackName),
		Status:       string(stack.StackStatus),
		StatusReason: aws.ToString(stack.StackStatusReason),
	}

	if stack.CreationTime != nil {
		result.CreationTime = *stack.CreationTime
	}
	if stack.LastUpdatedTime != nil {
		result.LastUpdatedTime = stack.LastUpdatedTime
	}

	// Extract outputs
	for _, out := range stack.Outputs {
		key := aws.ToString(out.OutputKey)
		value := aws.ToString(out.OutputValue)
		switch key {
		case "EcsOperatorServiceArn":
			result.EcsOperatorServiceArn = value
		case "EcsOperatorTaskDefinitionArn":
			result.EcsOperatorTaskDefinitionArn = value
		}
	}

	return result, nil
}

// StackEvent represents a single CloudFormation stack event.
type StackEvent struct {
	LogicalResourceID string
	ResourceType      string
	Status            string
	Reason            string
}

// GetFailedEvents returns stack events with CREATE_FAILED or UPDATE_FAILED status.
func GetFailedEvents(ctx context.Context, region, stackName string) ([]StackEvent, error) {
	client, err := newCFClient(ctx, region)
	if err != nil {
		return nil, err
	}

	output, err := client.DescribeStackEvents(ctx, &cloudformation.DescribeStackEventsInput{
		StackName: aws.String(stackName),
	})
	if err != nil {
		return nil, fmt.Errorf("describing stack events: %w", err)
	}

	var failed []StackEvent
	for _, event := range output.StackEvents {
		status := string(event.ResourceStatus)
		if status == string(cftypes.ResourceStatusCreateFailed) || status == string(cftypes.ResourceStatusUpdateFailed) {
			failed = append(failed, StackEvent{
				LogicalResourceID: aws.ToString(event.LogicalResourceId),
				ResourceType:      aws.ToString(event.ResourceType),
				Status:            status,
				Reason:            aws.ToString(event.ResourceStatusReason),
			})
		}
	}

	return failed, nil
}

// ForceNewDeployment forces a new deployment of an ECS service, pulling the latest image.
func ForceNewDeployment(ctx context.Context, region, cluster, serviceName string) error {
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return fmt.Errorf("loading AWS config: %w", err)
	}

	client := ecs.NewFromConfig(cfg)
	_, err = client.UpdateService(ctx, &ecs.UpdateServiceInput{
		Cluster:            aws.String(cluster),
		Service:            aws.String(serviceName),
		ForceNewDeployment: true,
	})
	if err != nil {
		return fmt.Errorf("forcing new deployment: %w", err)
	}

	return nil
}

// LogEntry represents a single log event from CloudWatch.
type LogEntry struct {
	Timestamp time.Time
	Message   string
	Stream    string
}

// GetRecentLogs fetches the most recent log events from a CloudWatch log group.
func GetRecentLogs(ctx context.Context, region, logGroup string, limit int) ([]LogEntry, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	client := cloudwatchlogs.NewFromConfig(cfg)

	// Get the most recent log streams
	streamsOutput, err := client.DescribeLogStreams(ctx, &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName: aws.String(logGroup),
		OrderBy:      "LastEventTime",
		Descending:   aws.Bool(true),
		Limit:        aws.Int32(5),
	})
	if err != nil {
		return nil, fmt.Errorf("describing log streams: %w", err)
	}

	if len(streamsOutput.LogStreams) == 0 {
		return nil, nil
	}

	// Collect stream names
	var streamNames []string
	for _, s := range streamsOutput.LogStreams {
		streamNames = append(streamNames, aws.ToString(s.LogStreamName))
	}

	input := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName:   aws.String(logGroup),
		LogStreamNames: streamNames,
		Limit:          aws.Int32(int32(limit)),
	}

	output, err := client.FilterLogEvents(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("filtering log events: %w", err)
	}

	var entries []LogEntry
	for _, event := range output.Events {
		entries = append(entries, LogEntry{
			Timestamp: time.UnixMilli(aws.ToInt64(event.Timestamp)),
			Message:   aws.ToString(event.Message),
			Stream:    aws.ToString(event.LogStreamName),
		})
	}

	return entries, nil
}
