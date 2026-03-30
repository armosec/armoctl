package ecs

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/armosec/armoctl/ecs/operator"
	"github.com/armosec/armoctl/ecs/patcher"
	"github.com/armosec/armoctl/internal/config"
	"github.com/armosec/armoctl/internal/version"
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

// sidecarConfig builds SidecarConfig from command flags and viper config.
func sidecarConfig(cmd *cobra.Command) patcher.SidecarConfig {
	agentImage, _ := cmd.Flags().GetString("agent-image")

	// Use cached version if no explicit image provided
	if agentImage == "" {
		agentImage = version.GetAgentImage()
	}

	return patcher.SidecarConfig{
		Image:        agentImage,
		CustomerGUID: viper.GetString("customer-guid"),
		AccessKey:    viper.GetString("access-key"),
	}
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
