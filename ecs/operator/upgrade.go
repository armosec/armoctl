package operator

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/armosec/armoctl/cmd/cliclient"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade the ARMO ECS Operator to the current image",
	Long: `Update the ARMO ECS Operator stack to the operator image ARMO currently serves
for the cluster's region (or --operator-image), keeping every other stack setting.

CloudFormation then rolls the operator service to the new image. If the stack
already runs that image, nothing changes.

Example:
  armoctl ecs operator upgrade -c arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster`,
	RunE: runUpgrade,
}

func init() {
	upgradeCmd.Flags().StringP("cluster", "c", "", "ECS cluster ARN (required)")
	upgradeCmd.Flags().String("operator-image", "", "ECS Operator container image (default: the image ARMO serves for the cluster region)")

	_ = upgradeCmd.MarkFlagRequired("cluster")

	OperatorCmd.AddCommand(upgradeCmd)
}

func runUpgrade(cmd *cobra.Command, args []string) error {
	if err := requireAuth(); err != nil {
		return err
	}
	ctx := cmd.Context()

	clusterARN, _ := cmd.Flags().GetString("cluster")
	cluster, err := parseClusterARN(clusterARN)
	if err != nil {
		return fmt.Errorf("invalid cluster ARN: %w", err)
	}
	stackName := defaultStackName(cluster.ClusterName)

	if _, err := DescribeStack(ctx, cluster.Region, stackName); err != nil {
		return fmt.Errorf("describing stack: %w", err)
	}

	image, err := operatorImageFor(cmd, cliclient.Default(viper.GetString)(cmd), cluster.Region)
	if err != nil {
		return err
	}

	cmd.PrintErrf("Upgrading ARMO ECS Operator...\n")
	cmd.PrintErrf("  Cluster: %s\n", cluster.ClusterName)
	cmd.PrintErrf("  Stack:   %s\n", stackName)
	cmd.PrintErrf("  Image:   %s\n", image)
	cmd.PrintErrln()

	changed, err := UpdateOperatorImage(ctx, cluster.Region, stackName, image)
	if err != nil {
		return err
	}
	if !changed {
		cmd.PrintErrln("The operator already runs this image; nothing to upgrade.")
		return nil
	}

	lastStatus := ""
	if _, err := WaitForStackUpdate(ctx, cluster.Region, stackName, func(status string) {
		if status != lastStatus {
			cmd.PrintErrf("  Status: %s\n", status)
			lastStatus = status
		}
	}); err != nil {
		return err
	}

	cmd.PrintErrln()
	cmd.PrintErrln("ARMO ECS Operator upgraded.")
	return nil
}
