// Package cliclient is the shared apiclient factory used by every cluster.
package cliclient

import (
	"strings"

	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/spf13/cobra"
)

// ClientFor returns the apiclient configured for the running command.
// Cluster commands take this as a function so tests can inject stubs.
type ClientFor func(cmd *cobra.Command) *apiclient.Client

// Default builds a ClientFor that reads viper config (via the supplied accessor).
// Uses api-base-url (NOT api-url, which is reserved for ECS/version-check).
func Default(read func(key string) string) ClientFor {
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
