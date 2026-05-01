package incidents

import (
	"strings"

	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/spf13/cobra"
)

func Cmd(clientFor ClientFor) *cobra.Command {
	c := &cobra.Command{Use: "incidents", Short: "Inspect and manage runtime incidents"}
	c.AddCommand(FieldsCmd())
	c.AddCommand(ListCmd(clientFor))
	c.AddCommand(AlertsCmd(clientFor))
	c.AddCommand(ExplainCmd(clientFor))
	c.AddCommand(ResolveCmd(clientFor))
	c.AddCommand(SeveritiesCmd(clientFor))
	return c
}

// DefaultClientFor reads viper config and builds an apiclient.
// It uses api-base-url (NOT api-url, which is reserved for ECS/version-check).
func DefaultClientFor(read func(key string) string) ClientFor {
	return func(cmd *cobra.Command) *apiclient.Client {
		base := read("api-base-url")
		// Strip any existing scheme to avoid double-prefixing.
		base = strings.TrimPrefix(base, "https://")
		base = strings.TrimPrefix(base, "http://")
		return apiclient.New(apiclient.Config{
			BaseURL:      "https://" + base,
			AccessKey:    read("access-key"),
			CustomerGUID: read("customer-guid"),
		})
	}
}
