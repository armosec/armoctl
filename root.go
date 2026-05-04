package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	ecscmd "github.com/armosec/armoctl/ecs"
	"github.com/armosec/armoctl/internal/config"
	"github.com/armosec/armoctl/internal/rootcmd"
	schemacmd "github.com/armosec/armoctl/internal/schema"
	versionpkg "github.com/armosec/armoctl/internal/version"
)

// updateCheckResult holds the result of the background version check.
var updateCheckResult chan *versionpkg.UpdateInfo

var rootCmd = &cobra.Command{
	Use:               "armoctl",
	Short:             "ARMO CLI for instrumenting cloud workloads",
	Long:              "armoctl is a CLI tool for instrumenting ECS task definitions with the ARMO runtime security agent.",
	PersistentPreRun:  startVersionCheck,
	PersistentPostRun: showUpdateBanner,
}

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure ARMO credentials",
	Long: `Set your Customer GUID and Access Key. Credentials are saved to ~/.armoctl/config.yaml.

By default 'configure' opens an interactive prompt. If any of the
--customer-guid, --access-key, --access-key-stdin, --api-base-url, or
--api-url flags is supplied, configure runs non-interactively and
treats a failed authentication ping as an error.

For scripts and AI-driven setup, prefer --access-key-stdin over
--access-key so the secret never appears in shell history or
'ps' output:

  echo "$ARMO_ACCESS_KEY" | armoctl configure \
      --customer-guid "$ARMO_CUSTOMER_GUID" \
      --access-key-stdin`,
	RunE: runConfigure,
}

func runConfigure(cmd *cobra.Command, _ []string) error {
	flagsUsed := false
	for _, name := range []string{"customer-guid", "access-key", "access-key-stdin", "api-base-url", "api-url"} {
		if cmd.Flags().Changed(name) {
			flagsUsed = true
			break
		}
	}
	if !flagsUsed {
		return config.PromptAllCredentials()
	}

	guid, _ := cmd.Flags().GetString("customer-guid")
	key, _ := cmd.Flags().GetString("access-key")
	apiBase, _ := cmd.Flags().GetString("api-base-url")
	apiURL, _ := cmd.Flags().GetString("api-url")

	if stdinFlag, _ := cmd.Flags().GetBool("access-key-stdin"); stdinFlag {
		if key != "" {
			return fmt.Errorf("--access-key and --access-key-stdin are mutually exclusive")
		}
		k, err := config.ReadAccessKeyFromStdin(cmd.InOrStdin())
		if err != nil {
			return err
		}
		if k == "" {
			return fmt.Errorf("--access-key-stdin: no value read from stdin")
		}
		key = k
	}

	return config.SaveCredentials(config.Credentials{
		CustomerGUID: guid,
		AccessKey:    key,
		APIBaseURL:   apiBase,
		APIURL:       apiURL,
	}, true)
}

func init() {
	cobra.OnInitialize(initConfig)

	configureCmd.Flags().String("customer-guid", "", "ARMO Customer GUID")
	configureCmd.Flags().String("access-key", "", "ARMO Access Key (avoid in shell history; prefer --access-key-stdin)")
	configureCmd.Flags().Bool("access-key-stdin", false, "Read the access key from stdin (recommended for scripts/AI agents)")
	configureCmd.Flags().String("api-base-url", "", "ARMO API base URL (default api.armosec.io)")
	configureCmd.Flags().String("api-url", "", "ARMO dashboard URL (default cloud.armosec.io)")

	rootCmd.AddCommand(ecscmd.EcsCmd)
	rootCmd.AddCommand(configureCmd)

	// We copy the factory's children + persistent flags onto our existing rootCmd
	// rather than using the factory's root directly, because rootCmd already has
	// PersistentPreRun/PersistentPostRun and the version-check hooks wired up.
	// Anything attached to the factory's root other than children and persistent
	// flags will NOT carry over — keep the factory minimal.
	built := rootcmd.NewRootCmd()
	for _, sub := range built.Commands() {
		rootCmd.AddCommand(sub)
	}
	rootCmd.PersistentFlags().AddFlagSet(built.PersistentFlags())

	rootCmd.AddCommand(schemacmd.Cmd())

	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug mode")
	_ = rootCmd.PersistentFlags().MarkHidden("debug")
	rootCmd.PersistentFlags().Bool("skip-update-check", false, "Skip checking for updates")
	_ = rootCmd.PersistentFlags().MarkHidden("skip-update-check")

	config.ApplyDefaults()

	_ = viper.BindEnv("api-url", "ARMO_API_URL")
	_ = viper.BindEnv("api-base-url", "ARMO_API_BASE_URL")
	_ = viper.BindEnv("customer-guid", "ARMO_CUSTOMER_GUID")
	_ = viper.BindEnv("access-key", "ARMO_ACCESS_KEY")
}

func initConfig() {
	home, err := os.UserHomeDir()
	if err == nil {
		configDir := filepath.Join(home, ".armoctl")
		viper.AddConfigPath(configDir)
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}
	_ = viper.ReadInConfig()
}

// startVersionCheck starts a background goroutine to check for updates.
func startVersionCheck(cmd *cobra.Command, args []string) {
	// Skip version check for certain commands
	if cmd.Name() == "version" || cmd.Name() == "update" {
		return
	}

	// Skip if user requested
	skipCheck, _ := cmd.Flags().GetBool("skip-update-check")
	if skipCheck {
		return
	}

	// Start background check
	updateCheckResult = make(chan *versionpkg.UpdateInfo, 1)
	go func() {
		defer close(updateCheckResult)

		latest, err := versionpkg.GetLatestVersions()
		if err != nil {
			return // Silently ignore errors
		}

		info := versionpkg.CheckForUpdates(Version, latest)
		updateCheckResult <- info
	}()
}

// showUpdateBanner displays the update banner if an update is available.
func showUpdateBanner(cmd *cobra.Command, args []string) {
	if updateCheckResult == nil {
		return
	}

	// Wait briefly for the check to complete
	select {
	case info := <-updateCheckResult:
		versionpkg.PrintUpdateBanner(info)
	case <-time.After(1 * time.Second):
		// Don't block the user for too long
	}
}
