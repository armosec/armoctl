package repoposture

import (
	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/cmd/cliflags"
	"github.com/armosec/armoctl/internal/output"
	"github.com/spf13/cobra"
)

func FailedControlsCmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{
		Use:   "failed-controls",
		Short: "List failed controls per repo",
		RunE: func(cmd *cobra.Command, args []string) error {
			cli := clientFor(cmd)
			reportGUID, _ := cmd.Flags().GetString("report-guid")
			kind, _ := cmd.Flags().GetString("kind")
			body := map[string]any{
				"reportGUID": reportGUID,
				"kind":       kind,
			}
			var result []any
			if err := cli.PostJSON(cmd.Context(), "/repositoryPosture/failedControls", nil, body, &result); err != nil {
				return err
			}
			list := output.List{Items: result, Total: len(result)}
			return output.Render(cmd.OutOrStdout(), list, cliflags.OutputOptions(cmd, FailedControlSummary))
		},
	}
	c.Flags().String("report-guid", "", "Report GUID (required)")
	c.Flags().String("kind", "repo", "Entity kind: repo or file")
	_ = c.MarkFlagRequired("report-guid")
	return c
}
