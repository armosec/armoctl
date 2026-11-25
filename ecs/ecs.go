package ecs

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/ecs/operator"
	"github.com/armosec/armoctl/ecs/patcher"
	"github.com/armosec/armoctl/internal/config"
	"github.com/armosec/armoctl/internal/sensors"
)

// Test seams for the backend image lookup and AWS region detection.
var (
	fetchSensors = sensors.Fetch
	regionFor    = awsRegion
)

var EcsCmd = &cobra.Command{
	Use:     "ecs",
	Short:   "Operate on ECS resources",
	Aliases: []string{"e"},
}

func init() {
	EcsCmd.PersistentFlags().StringSlice("container", nil, "Container names to patch (repeatable; default: all)")
	EcsCmd.PersistentFlags().String("agent-image", "", "Agent sidecar image")
	EcsCmd.PersistentFlags().Bool("volume-fixer", false, "Include a volume-fixer init container to chmod the shared volume")

	EcsCmd.AddCommand(operator.OperatorCmd)
}

// patchOpts builds PatchOptions from command flags.
func patchOpts(cmd *cobra.Command) patcher.PatchOptions {
	containers, _ := cmd.Flags().GetStringSlice("container")
	volumeFixer, _ := cmd.Flags().GetBool("volume-fixer")
	return patcher.PatchOptions{
		Containers:  containers,
		VolumeFixer: volumeFixer,
	}
}

// regionFromARN returns the region of an AWS ARN, or "" when s is not an ARN.
func regionFromARN(s string) string {
	parsed, err := arn.Parse(s)
	if err != nil {
		return ""
	}
	return parsed.Region
}

// sidecarConfig builds SidecarConfig from command flags and viper config.
// Without --agent-image, the image comes from the ARMO backend for
// regionHint (the region of a task-definition or cluster ARN the command
// was given), or else the region of the AWS config.
func sidecarConfig(cmd *cobra.Command, regionHint string) (patcher.SidecarConfig, error) {
	agentImage, _ := cmd.Flags().GetString("agent-image")

	if agentImage == "" {
		if err := requireAuth(); err != nil {
			return patcher.SidecarConfig{}, err
		}
		ctx := cmd.Context()
		region := regionHint
		if region == "" {
			var err error
			if region, err = regionFor(ctx); err != nil {
				return patcher.SidecarConfig{}, fmt.Errorf("cannot pick the agent image: %w; set AWS_REGION (or a region in your AWS profile), or pass --agent-image <image>", err)
			}
		}
		versions, err := fetchSensors(ctx, cliclient.Default(viper.GetString)(cmd), region)
		if err != nil {
			return patcher.SidecarConfig{}, fmt.Errorf("%w; or pass --agent-image <image>", err)
		}
		if agentImage, err = versions.AgentImage(); err != nil {
			return patcher.SidecarConfig{}, err
		}
	}

	return patcher.SidecarConfig{
		Image:        agentImage,
		CustomerGUID: viper.GetString("customer-guid"),
		AccessKey:    viper.GetString("access-key"),
	}, nil
}

// patchAndPrint patches a task definition and prints the JSON result to stdout.
func patchAndPrint(cmd *cobra.Command, td *patcher.TaskDefinition, opts patcher.PatchOptions, sidecar patcher.SidecarConfig) error {
	if err := patcher.Patch(td, opts, sidecar); err != nil {
		return fmt.Errorf("patching task definition: %w", err)
	}
	out, err := patcher.MarshalTaskDef(td)
	if err != nil {
		return fmt.Errorf("marshaling output: %w", err)
	}
	cmd.Println(string(out))
	return nil
}

// requireAuth checks credentials, prompting interactively if missing.
func requireAuth() error {
	return config.RequireAuth()
}
