package main

import (
	"fmt"
	"os"
	"os/exec"

	versionpkg "github.com/armosec/armoctl/internal/version"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update armoctl to the latest version",
	Long:  "Download and install the latest version of armoctl, replacing the current binary.",
	RunE:  runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	// Fetch latest version info
	fmt.Println("Checking for updates...")

	latest, err := versionpkg.FetchLatest()
	if err != nil {
		return fmt.Errorf("checking for updates: %w", err)
	}

	// Check if update is needed
	info := versionpkg.CheckForUpdates(Version, latest)
	if !info.HasUpdate {
		fmt.Printf("armoctl is already up to date (version %s)\n", Version)
		return nil
	}

	fmt.Printf("Current version: %s\n", Version)
	fmt.Printf("Latest version:  %s\n", latest.Armoctl)
	fmt.Println()

	// Show where the binary is located
	execPath, err := versionpkg.GetExecutablePath()
	if err != nil {
		return fmt.Errorf("getting executable path: %w", err)
	}
	fmt.Printf("Binary location: %s\n", execPath)
	fmt.Println()

	// Perform the update
	fmt.Println("Downloading update...")
	if err := versionpkg.SelfUpdate(); err != nil {
		return fmt.Errorf("updating: %w", err)
	}

	fmt.Println()
	fmt.Printf("Successfully updated to %s\n", latest.Armoctl)

	// Verify the update by running version command
	fmt.Println()
	fmt.Println("Verifying installation:")
	verifyCmd := exec.Command(execPath, "version")
	verifyCmd.Stdout = os.Stdout
	verifyCmd.Stderr = os.Stderr
	_ = verifyCmd.Run() // Ignore error, just for display

	return nil
}
