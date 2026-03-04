package ecs

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/spf13/cobra"

	"github.com/armosec/armoctl/ecs/patcher"
)

var patchCmd = &cobra.Command{
	Use:     "patch <taskdef>",
	Short:   "Patch an ECS task definition with the ARMO runtime agent",
	Long:    "Patch an ECS task definition JSON with the ARMO ptrace sidecar.\nSource can be a file path, an ARN (starts with \"arn:\"), or \"-\" for stdin.",
	Aliases: []string{"p"},
	Args:    cobra.ExactArgs(1),
	RunE:    runPatch,
}

func init() {
	patchCmd.Flags().Bool("register", false, "Register the patched task definition with AWS")

	EcsCmd.AddCommand(patchCmd)
}

func runPatch(cmd *cobra.Command, args []string) error {
	register, _ := cmd.Flags().GetBool("register")
	if register {
		if err := requireAuth(); err != nil {
			return err
		}
	}

	td, err := loadTaskDef(cmd.Context(), args[0])
	if err != nil {
		return fmt.Errorf("loading task definition: %w", err)
	}

	if err := patchAndPrint(td, patchOpts(cmd), sidecarConfig(cmd)); err != nil {
		return err
	}

	if register {
		client, err := newECSClient(cmd.Context())
		if err != nil {
			return err
		}
		if _, err := registerTaskDef(cmd.Context(), client, td); err != nil {
			return fmt.Errorf("registering task definition: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Registered task definition: %s\n", aws.ToString(td.Family))
	}

	return nil
}

// loadTaskDef loads a TaskDefinition from a file path, ARN, or stdin ("-").
func loadTaskDef(ctx context.Context, source string) (*patcher.TaskDefinition, error) {
	if source == "-" {
		return loadTaskDefFromReader(os.Stdin)
	}

	if strings.HasPrefix(source, "arn:") {
		return loadTaskDefFromARN(ctx, source)
	}

	f, err := os.Open(source)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return loadTaskDefFromReader(f)
}

// loadTaskDefFromReader decodes a TaskDefinition from an io.Reader.
func loadTaskDefFromReader(r io.Reader) (*patcher.TaskDefinition, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}
	return patcher.UnmarshalTaskDef(data)
}

// loadTaskDefFromARN fetches a task definition from AWS using its ARN.
func loadTaskDefFromARN(ctx context.Context, arn string) (*patcher.TaskDefinition, error) {
	client, err := newECSClient(ctx)
	if err != nil {
		return nil, err
	}
	return describeTaskDef(ctx, client, arn)
}
