package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	ecscmd "github.com/armosec/armoctl/ecs"
	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/cmd/cliflags"
	cloudaccountscmd "github.com/armosec/armoctl/cmd/cloudaccounts"
	incidentscmd "github.com/armosec/armoctl/cmd/incidents"
	networkpoliciescmd "github.com/armosec/armoctl/cmd/networkpolicies"
	posturecmd "github.com/armosec/armoctl/cmd/posture"
	seccompcmd "github.com/armosec/armoctl/cmd/seccomp"
	vulnscmd "github.com/armosec/armoctl/cmd/vulns"
	"github.com/armosec/armoctl/internal/config"
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
	Long:  "Interactively set your Customer GUID and Access Key. Credentials are saved to ~/.armoctl/config.yaml.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return config.PromptAllCredentials()
	},
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.AddCommand(ecscmd.EcsCmd)
	rootCmd.AddCommand(configureCmd)

	cliflags.Register(rootCmd)
	rootCmd.AddCommand(incidentscmd.Cmd(cliclient.Default(viper.GetString)))
	rootCmd.AddCommand(vulnscmd.Cmd(cliclient.Default(viper.GetString)))
	rootCmd.AddCommand(posturecmd.Cmd(cliclient.Default(viper.GetString)))
	rootCmd.AddCommand(networkpoliciescmd.Cmd(cliclient.Default(viper.GetString)))
	rootCmd.AddCommand(seccompcmd.Cmd(cliclient.Default(viper.GetString)))
	rootCmd.AddCommand(cloudaccountscmd.Cmd(cliclient.Default(viper.GetString)))
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
