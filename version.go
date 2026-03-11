package main

import (
	"runtime"

	"github.com/spf13/cobra"
)

// Set by goreleaser / ldflags at build time
var (
	// Version is the current armoctl version, set at build time.
	Version   = "dev"
	commit    = "none"
	buildTime = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println("armoctl")
		cmd.Println()
		cmd.Printf("  Version:     %s\n", Version)
		cmd.Printf("  Go Version:  %s\n", runtime.Version())
		cmd.Printf("  OS/Arch:     %s (%s)\n", runtime.GOOS, runtime.GOARCH)
		cmd.Printf("  Build Time:  %s\n", buildTime)
		cmd.Printf("  Commit SHA:  %s\n", commit)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
