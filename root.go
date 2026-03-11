package main

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	ecscmd "github.com/armosec/armoctl/ecs"
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

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.AddCommand(ecscmd.EcsCmd)

	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug mode")
	rootCmd.PersistentFlags().Bool("skip-update-check", false, "Skip checking for updates")
	rootCmd.PersistentFlags().String("api-url", "cloud.armosec.io", "ARMO platform URL")
	rootCmd.PersistentFlags().String("customer-guid", "", "Customer GUID")
	rootCmd.PersistentFlags().String("access-key", "", "API access key")

	_ = viper.BindPFlag("api-url", rootCmd.PersistentFlags().Lookup("api-url"))
	_ = viper.BindPFlag("customer-guid", rootCmd.PersistentFlags().Lookup("customer-guid"))
	_ = viper.BindPFlag("access-key", rootCmd.PersistentFlags().Lookup("access-key"))

	_ = viper.BindEnv("api-url", "ARMO_API_URL")
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

	// Wait for the check to complete (should be instant if already done)
	select {
	case info := <-updateCheckResult:
		versionpkg.PrintUpdateBanner(info)
	default:
		// Check not complete yet, don't block
	}
}
