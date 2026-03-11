package version

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

// PrintUpdateBanner prints an update notification banner to stderr.
// Only prints if stdout is a terminal (not piped).
func PrintUpdateBanner(info *UpdateInfo) {
	if info == nil || !info.HasUpdate {
		return
	}

	// Don't print banner if output is being piped
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return
	}

	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "┌────────────────────────────────────────────────────────┐")
	fmt.Fprintf(os.Stderr, "│  Update available: %s -> %s%s│\n",
		info.ArmoCtlCurrent, info.ArmoCtlLatest, padding(info.ArmoCtlCurrent, info.ArmoCtlLatest))
	fmt.Fprintln(os.Stderr, "│                                                        │")
	fmt.Fprintln(os.Stderr, "│  Run: armoctl update                                   │")
	fmt.Fprintln(os.Stderr, "└────────────────────────────────────────────────────────┘")
}

// padding calculates spaces needed to align the box border.
func padding(current, latest string) string {
	// Box inner width is 54 chars
	// "  Update available: " = 20 chars
	// " -> " = 4 chars
	// We need to pad to reach 54
	contentLen := 20 + len(current) + 4 + len(latest)
	padLen := 54 - contentLen
	if padLen < 1 {
		padLen = 1
	}
	spaces := make([]byte, padLen)
	for i := range spaces {
		spaces[i] = ' '
	}
	return string(spaces)
}
