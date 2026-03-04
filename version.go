package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Set by goreleaser / ldflags at build time
var (
	version   = "dev"
	commit    = "none"
	buildTime = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("armoctl\n\n")
		fmt.Printf("  Version:     %s\n", version)
		fmt.Printf("  Go Version:  %s\n", runtime.Version())
		fmt.Printf("  OS/Arch:     %s (%s)\n", runtime.GOOS, runtime.GOARCH)
		fmt.Printf("  Build Time:  %s\n", buildTime)
		fmt.Printf("  Commit SHA:  %s\n", commit)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
