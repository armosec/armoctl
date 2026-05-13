package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	cftypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
)

type stackParams struct {
	StackName         string
	Region            string
	ClusterName       string
	CustomerGUID      string
	AccessKey         string
	APIUrl            string
	CloudWatchLogs    string
	MaxLearningPeriod string
}

type stackOutput struct {
	StackName       string
	Status          string
	StatusReason    string
	CreationTime    time.Time
	LastUpdatedTime *time.Time
	Outputs         map[string]string
}

type stackEvent struct {
	LogicalResourceID string
	ResourceType      string
	Reason            string
}

type logEntry struct {
	Timestamp time.Time
	Message   string
}

func newCFClient(ctx context.Context, region string) (*cloudformation.Client, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}
	return cloudformation.NewFromConfig(cfg), nil
}

func createStack(ctx context.Context, params stackParams) error {
	client, err := newCFClient(ctx, params.Region)
	if err != nil {
		return err
	}

	exists, err := stackExists(ctx, client, params.StackName)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("stack %q already exists. Use 'armoctl ecs agent uninstall' first", params.StackName)
	}

	cfParams := []cftypes.Parameter{
		{ParameterKey: aws.String("Region"), ParameterValue: aws.String(params.Region)},
		{ParameterKey: aws.String("ClusterName"), ParameterValue: aws.String(params.ClusterName)},
		{ParameterKey: aws.String("CustomerGUID"), ParameterValue: aws.String(params.CustomerGUID)},
		{ParameterKey: aws.String("AccessKey"), ParameterValue: aws.String(params.AccessKey)},
		{ParameterKey: aws.String("ApiUrl"), ParameterValue: aws.String(params.APIUrl)},
		{ParameterKey: aws.String("CloudWatchLogsGroupName"), ParameterValue: aws.String(params.CloudWatchLogs)},
	}
	if params.MaxLearningPeriod != "" {
		cfParams = append(cfParams, cftypes.Parameter{
			ParameterKey:   aws.String("MaxLearningPeriod"),
			ParameterValue: aws.String(params.MaxLearningPeriod),
		})
	}

	_, err = client.CreateStack(ctx, &cloudformation.CreateStackInput{
		StackName:    aws.String(params.StackName),
		TemplateURL:  aws.String(TemplateURL),
		Parameters:   cfParams,
		Capabilities: []cftypes.Capability{cftypes.CapabilityCapabilityNamedIam},
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

func waitForStackCreate(ctx context.Context, region, stackName string, progress func(string)) (*stackOutput, error) {
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
			out, err := describeStack(ctx, client, stackName)
			if err != nil {
				return nil, err
			}
			if progress != nil {
				progress(out.Status)
			}
			switch out.Status {
			case string(cftypes.StackStatusCreateComplete):
				return out, nil
			case string(cftypes.StackStatusCreateFailed),
				string(cftypes.StackStatusRollbackComplete),
				string(cftypes.StackStatusRollbackFailed):
				reason := out.StatusReason
				if reason == "" {
					reason = "check CloudFormation console for details"
				}
				return out, fmt.Errorf("stack creation failed: %s - %s", out.Status, reason)
			}
		}
	}
}

func deleteStack(ctx context.Context, region, stackName string) error {
	client, err := newCFClient(ctx, region)
	if err != nil {
		return err
	}

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

func waitForStackDelete(ctx context.Context, region, stackName string, progress func(string)) error {
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
			out, err := describeStack(ctx, client, stackName)
			if err != nil {
				var notFound *cftypes.StackNotFoundException
				if errors.As(err, &notFound) {
					return nil
				}
				return err
			}
			if progress != nil {
				progress(out.Status)
			}
			switch out.Status {
			case string(cftypes.StackStatusDeleteComplete):
				return nil
			case string(cftypes.StackStatusDeleteFailed):
				reason := out.StatusReason
				if reason == "" {
					reason = "check CloudFormation console for details"
				}
				return fmt.Errorf("stack deletion failed: %s", reason)
			}
		}
	}
}

func describeStackByName(ctx context.Context, region, stackName string) (*stackOutput, error) {
	client, err := newCFClient(ctx, region)
	if err != nil {
		return nil, err
	}
	return describeStack(ctx, client, stackName)
}

func stackExists(ctx context.Context, client *cloudformation.Client, stackName string) (bool, error) {
	out, err := client.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	})
	if err != nil {
		var notFound *cftypes.StackNotFoundException
		if errors.As(err, &notFound) {
			return false, nil
		}
		if isStackNotFoundError(err) {
			return false, nil
		}
		return false, fmt.Errorf("describing stack: %w", err)
	}
	if len(out.Stacks) == 0 {
		return false, nil
	}
	if out.Stacks[0].StackStatus == cftypes.StackStatusDeleteComplete {
		return false, nil
	}
	return true, nil
}

func isStackNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "does not exist") || strings.Contains(msg, "not found")
}

func describeStack(ctx context.Context, client *cloudformation.Client, stackName string) (*stackOutput, error) {
	out, err := client.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
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
	if len(out.Stacks) == 0 {
		return nil, &cftypes.StackNotFoundException{Message: aws.String("stack not found")}
	}

	stack := out.Stacks[0]
	result := &stackOutput{
		StackName:    aws.ToString(stack.StackName),
		Status:       string(stack.StackStatus),
		StatusReason: aws.ToString(stack.StackStatusReason),
		Outputs:      make(map[string]string),
	}
	if stack.CreationTime != nil {
		result.CreationTime = *stack.CreationTime
	}
	if stack.LastUpdatedTime != nil {
		result.LastUpdatedTime = stack.LastUpdatedTime
	}
	for _, o := range stack.Outputs {
		result.Outputs[aws.ToString(o.OutputKey)] = aws.ToString(o.OutputValue)
	}
	return result, nil
}

func getFailedEvents(ctx context.Context, region, stackName string) ([]stackEvent, error) {
	client, err := newCFClient(ctx, region)
	if err != nil {
		return nil, err
	}

	out, err := client.DescribeStackEvents(ctx, &cloudformation.DescribeStackEventsInput{
		StackName: aws.String(stackName),
	})
	if err != nil {
		return nil, fmt.Errorf("describing stack events: %w", err)
	}

	var failed []stackEvent
	for _, event := range out.StackEvents {
		status := string(event.ResourceStatus)
		if status == string(cftypes.ResourceStatusCreateFailed) || status == string(cftypes.ResourceStatusUpdateFailed) {
			failed = append(failed, stackEvent{
				LogicalResourceID: aws.ToString(event.LogicalResourceId),
				ResourceType:      aws.ToString(event.ResourceType),
				Reason:            aws.ToString(event.ResourceStatusReason),
			})
		}
	}
	return failed, nil
}

func getRecentLogs(ctx context.Context, region, logGroup string, limit int) ([]logEntry, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}
	client := cloudwatchlogs.NewFromConfig(cfg)

	streamsOut, err := client.DescribeLogStreams(ctx, &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName: aws.String(logGroup),
		OrderBy:      "LastEventTime",
		Descending:   aws.Bool(true),
		Limit:        aws.Int32(5),
	})
	if err != nil {
		return nil, fmt.Errorf("describing log streams: %w", err)
	}
	if len(streamsOut.LogStreams) == 0 {
		return nil, nil
	}

	var streamNames []string
	for _, s := range streamsOut.LogStreams {
		streamNames = append(streamNames, aws.ToString(s.LogStreamName))
	}

	eventsOut, err := client.FilterLogEvents(ctx, &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName:   aws.String(logGroup),
		LogStreamNames: streamNames,
		Limit:          aws.Int32(int32(limit)),
	})
	if err != nil {
		return nil, fmt.Errorf("filtering log events: %w", err)
	}

	var entries []logEntry
	for _, event := range eventsOut.Events {
		entries = append(entries, logEntry{
			Timestamp: time.UnixMilli(aws.ToInt64(event.Timestamp)),
			Message:   aws.ToString(event.Message),
		})
	}
	return entries, nil
}
