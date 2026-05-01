package integrations

import (
	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/spf13/cobra"
)

func Cmd(clientFor cliclient.ClientFor) *cobra.Command {
	c := &cobra.Command{Use: "integrations", Short: "Manage Jira/SIEM/alert-channel integrations"}
	c.AddCommand(FieldsCmd())

	jira := &cobra.Command{Use: "jira", Short: "Jira integration commands"}
	jira.AddCommand(JiraProjectsCmd(clientFor))
	jira.AddCommand(JiraIssueTypesCmd(clientFor))
	jira.AddCommand(JiraFieldsCmd(clientFor))
	jira.AddCommand(JiraCreateTicketCmd(clientFor))
	c.AddCommand(jira)

	ac := &cobra.Command{Use: "alert-channels", Short: "Alert channel integrations"}
	ac.AddCommand(AlertChannelsCreateCmd(clientFor))
	c.AddCommand(ac)

	siem := &cobra.Command{Use: "siem", Short: "SIEM integrations"}
	siem.AddCommand(SiemCreateCmd(clientFor))
	c.AddCommand(siem)

	c.AddCommand(UnlinkCmd(clientFor))
	return c
}
