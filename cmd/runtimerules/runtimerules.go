// Package runtimerules implements the `armoctl runtime-rules` cluster.
package runtimerules

import (
	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/spf13/cobra"
)

func Cmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{Use: "runtime-rules", Short: "Manage and evaluate runtime rules"}
	c.AddCommand(FieldsCmd())
	c.AddCommand(ListCmd(clientFor))
	c.AddCommand(GetCmd(clientFor))
	c.AddCommand(CreateCmd(clientFor))
	c.AddCommand(UpdateCmd(clientFor))
	c.AddCommand(DeleteCmd(clientFor))
	c.AddCommand(EvaluateCmd(clientFor))
	return c
}
