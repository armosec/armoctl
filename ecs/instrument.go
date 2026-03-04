package ecs

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsecs "github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/spf13/cobra"

	"github.com/armosec/armoctl/ecs/patcher"
)

var instrumentCmd = &cobra.Command{
	Use:     "instrument",
	Short:   "Instrument a live ECS service with the ARMO runtime agent",
	Long:    "Fetch the current task definition from a running ECS service, patch it with the ARMO ptrace sidecar, and optionally register and deploy.",
	Aliases: []string{"i"},
	RunE:    runInstrument,
}

func init() {
	instrumentCmd.Flags().StringP("cluster", "c", "", "ECS cluster name or ARN (required)")
	instrumentCmd.Flags().StringP("service", "s", "", "ECS service name or ARN (required)")
	instrumentCmd.Flags().Bool("deploy", false, "Register the patched task definition and update the service")

	_ = instrumentCmd.MarkFlagRequired("cluster")
	_ = instrumentCmd.MarkFlagRequired("service")

	EcsCmd.AddCommand(instrumentCmd)
}

func runInstrument(cmd *cobra.Command, args []string) error {
	deploy, _ := cmd.Flags().GetBool("deploy")
	if deploy {
		if err := requireAuth(); err != nil {
			return err
		}
	}

	cluster, _ := cmd.Flags().GetString("cluster")
	service, _ := cmd.Flags().GetString("service")
	ctx := cmd.Context()

	td, err := fetchServiceTaskDef(ctx, cluster, service)
	if err != nil {
		return fmt.Errorf("fetching service task definition: %w", err)
	}

	if err := patchAndPrint(td, patchOpts(cmd), sidecarConfig(cmd)); err != nil {
		return err
	}

	if deploy {
		newArn, err := registerAndUpdate(ctx, td, cluster, service)
		if err != nil {
			return fmt.Errorf("registering and updating service: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Registered task definition: %s\n", newArn)
		fmt.Fprintf(os.Stderr, "Updated service %s in cluster %s\n", service, cluster)
	}

	return nil
}

// fetchServiceTaskDef fetches the current task definition for an ECS service.
func fetchServiceTaskDef(ctx context.Context, cluster, service string) (*patcher.TaskDefinition, error) {
	client, err := newECSClient(ctx)
	if err != nil {
		return nil, err
	}

	svcOut, err := client.DescribeServices(ctx, &awsecs.DescribeServicesInput{
		Cluster:  aws.String(cluster),
		Services: []string{service},
	})
	if err != nil {
		return nil, fmt.Errorf("describing service: %w", err)
	}
	if len(svcOut.Services) == 0 {
		return nil, fmt.Errorf("service %q not found in cluster %q", service, cluster)
	}
	taskDefArn := svcOut.Services[0].TaskDefinition
	if taskDefArn == nil || *taskDefArn == "" {
		return nil, fmt.Errorf("service %q has no task definition", service)
	}

	return describeTaskDef(ctx, client, *taskDefArn)
}

// registerAndUpdate registers a new task definition and updates the service to use it.
func registerAndUpdate(ctx context.Context, td *patcher.TaskDefinition, cluster, service string) (string, error) {
	client, err := newECSClient(ctx)
	if err != nil {
		return "", err
	}

	newArn, err := registerTaskDef(ctx, client, td)
	if err != nil {
		return "", err
	}

	_, err = client.UpdateService(ctx, &awsecs.UpdateServiceInput{
		Cluster:        aws.String(cluster),
		Service:        aws.String(service),
		TaskDefinition: aws.String(newArn),
	})
	if err != nil {
		return newArn, fmt.Errorf("calling UpdateService: %w", err)
	}

	return newArn, nil
}
