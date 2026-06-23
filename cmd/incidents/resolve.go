package incidents

import (
	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/spf13/cobra"
)

func ResolveCmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{
		Use:   "resolve [guid]",
		Short: "Resolve a runtime incident (sets status to Resolved)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return &clierr.Error{Code: clierr.CodeBadInput, Msg: "resolve requires a GUID"}
			}
			falsePositive, _ := cmd.Flags().GetBool("false-positive")
			return runStatusChange(cmd, clientFor(cmd), statusChangeOpts{
				status:        "Resolved",
				guids:         []string{args[0]},
				falsePositive: falsePositive,
				commandName:   "incidents.resolve",
			})
		},
	}
	c.Flags().Bool("false-positive", false, "Mark the incident as a false positive when resolving")
	return c
}
