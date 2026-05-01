package repoposture

import (
	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/spf13/cobra"
)

func Cmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{Use: "repo-posture", Short: "Inspect repository (IaC) posture"}
	c.AddCommand(FieldsCmd())
	c.AddCommand(RepositoriesCmd(clientFor))
	c.AddCommand(FilesCmd(clientFor))
	c.AddCommand(ResourcesCmd(clientFor))
	c.AddCommand(FailedControlsCmd(clientFor))
	return c
}
