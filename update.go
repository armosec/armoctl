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
	// Fetch latest version info from the binary distribution CDN.
	// 'armoctl update' deliberately does not depend on configured
	// credentials — anyone should be able to upgrade without first
	// running 'armoctl configure'.
	cmd.Println("Checking for updates...")

	latest, err := versionpkg.FetchLatestArmoctl(cmd.Context())
	if err != nil {
		return fmt.Errorf("checking for updates: %w", err)
	}

	// Check if update is needed
	info := versionpkg.CheckForUpdates(Version, latest)
	if !info.HasUpdate {
		cmd.Printf("armoctl is already up to date (version %s)\n", Version)
		return nil
	}

	cmd.Printf("Current version: %s\n", Version)
	cmd.Printf("Latest version:  %s\n", latest)
	cmd.Println()

	// Show where the binary is located
	execPath, err := versionpkg.GetExecutablePath()
	if err != nil {
		return fmt.Errorf("getting executable path: %w", err)
	}
	cmd.Printf("Binary location: %s\n", execPath)
	cmd.Println()

	// Perform the update
	cmd.Println("Downloading update...")
	if err := versionpkg.SelfUpdate(); err != nil {
		return fmt.Errorf("updating: %w", err)
	}

	cmd.Println()
	cmd.Printf("Successfully updated to %s\n", latest)

	// Verify the update by running version command
	cmd.Println()
	cmd.Println("Verifying installation:")
	verifyCmd := exec.Command(execPath, "version")
	verifyCmd.Stdout = os.Stdout
	verifyCmd.Stderr = os.Stderr
	if err := verifyCmd.Run(); err != nil {
		cmd.PrintErrf("warning: failed to verify updated binary: %v\n", err)
	}

	return nil
}
