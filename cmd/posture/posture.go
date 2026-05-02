package posture

import (
	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/spf13/cobra"
)

func Cmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{Use: "posture", Short: "Inspect compliance posture (frameworks, controls, resources)"}
	c.AddCommand(FieldsCmd())
	c.AddCommand(FrameworksCmd(clientFor))
	c.AddCommand(ControlsCmd(clientFor))
	c.AddCommand(ResourcesCmd(clientFor))

	exc := &cobra.Command{Use: "exceptions", Short: "Posture exception policies"}
	exc.AddCommand(ExceptionsListCmd(clientFor))
	exc.AddCommand(ExceptionsCreateCmd(clientFor))
	exc.AddCommand(ExceptionsDeleteCmd(clientFor))
	c.AddCommand(exc)

	return c
}
