// Package rootcmd builds the cobra command tree. main() and the gen-skill-docs
// generator both call NewRootCmd so they see exactly the same tree.
package rootcmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	attackchainscmd "github.com/armosec/armoctl/cmd/attackchains"
	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/cmd/cliflags"
	cloudaccountscmd "github.com/armosec/armoctl/cmd/cloudaccounts"
	incidentscmd "github.com/armosec/armoctl/cmd/incidents"
	integrationscmd "github.com/armosec/armoctl/cmd/integrations"
	inventorycmd "github.com/armosec/armoctl/cmd/inventory"
	networkpoliciescmd "github.com/armosec/armoctl/cmd/networkpolicies"
	posturecmd "github.com/armosec/armoctl/cmd/posture"
	repoposturecmd "github.com/armosec/armoctl/cmd/repoposture"
	riskscmd "github.com/armosec/armoctl/cmd/risks"
	runtimepoliciescmd "github.com/armosec/armoctl/cmd/runtimepolicies"
	runtimerulescmd "github.com/armosec/armoctl/cmd/runtimerules"
	seccompcmd "github.com/armosec/armoctl/cmd/seccomp"
	vulnscmd "github.com/armosec/armoctl/cmd/vulns"
)

// NewRootCmd builds and returns the configured root cobra command,
// populated with every cluster subcommand. It does NOT register the
// ECS, configure, schema, or version-check infrastructure that main()
// adds — those concerns belong to main.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "armoctl",
		Short: "ARMO CLI for instrumenting cloud workloads",
		Long:  "armoctl is a CLI tool for instrumenting ECS task definitions with the ARMO runtime security agent.",
	}

	cliflags.Register(root)
	cf := cliclient.Default(viper.GetString)
	root.AddCommand(incidentscmd.Cmd(cf))
	root.AddCommand(vulnscmd.Cmd(cf))
	root.AddCommand(posturecmd.Cmd(cf))
	root.AddCommand(riskscmd.Cmd(cf))
	root.AddCommand(attackchainscmd.Cmd(cf))
	root.AddCommand(inventorycmd.Cmd(cf))
	root.AddCommand(networkpoliciescmd.Cmd(cf))
	root.AddCommand(seccompcmd.Cmd(cf))
	root.AddCommand(cloudaccountscmd.Cmd(cf))
	root.AddCommand(runtimerulescmd.Cmd(cf))
	root.AddCommand(runtimepoliciescmd.Cmd(cf))
	root.AddCommand(integrationscmd.Cmd(cf))
	root.AddCommand(repoposturecmd.Cmd(cf))

	return root
}
