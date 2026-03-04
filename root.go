package main

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	ecscmd "github.com/armosec/armoctl/ecs"
)

var rootCmd = &cobra.Command{
	Use:   "armoctl",
	Short: "ARMO CLI for instrumenting cloud workloads",
	Long:  "armoctl is a CLI tool for instrumenting ECS task definitions with the ARMO runtime security agent.",
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.AddCommand(ecscmd.EcsCmd)

	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug mode")
	rootCmd.PersistentFlags().String("api-url", "cloud.armosec.io", "ARMO platform URL")
	rootCmd.PersistentFlags().String("customer-guid", "", "Customer GUID")
	rootCmd.PersistentFlags().String("access-key", "", "API access key")

	_ = viper.BindPFlag("api-url", rootCmd.PersistentFlags().Lookup("api-url"))
	_ = viper.BindPFlag("customer-guid", rootCmd.PersistentFlags().Lookup("customer-guid"))
	_ = viper.BindPFlag("access-key", rootCmd.PersistentFlags().Lookup("access-key"))

	viper.BindEnv("api-url", "ARMO_API_URL")
	viper.BindEnv("customer-guid", "ARMO_CUSTOMER_GUID")
	viper.BindEnv("access-key", "ARMO_ACCESS_KEY")
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
