package incidents

import (
	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/spf13/cobra"
)

func Cmd(clientFor ClientFor) *cobra.Command {
	c := &cobra.Command{Use: "incidents", Short: "Inspect and manage runtime incidents"}
	c.AddCommand(FieldsCmd())
	c.AddCommand(ListCmd(clientFor))
	c.AddCommand(GetCmd(clientFor))
	return c
}

// DefaultClientFor reads viper config and builds an apiclient.
// It uses api-base-url (NOT api-url, which is reserved for ECS/version-check).
func DefaultClientFor(read func(key string) string) ClientFor {
	return func(cmd *cobra.Command) *apiclient.Client {
		return apiclient.New(apiclient.Config{
			BaseURL:      "https://" + read("api-base-url"),
			AccessKey:    read("access-key"),
			CustomerGUID: read("customer-guid"),
		})
	}
}
