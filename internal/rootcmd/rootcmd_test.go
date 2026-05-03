package rootcmd_test

import (
	"testing"

	"github.com/armosec/armoctl/internal/rootcmd"
)

func TestNewRootCmdHasAllClusters(t *testing.T) {
	root := rootcmd.NewRootCmd()
	want := []string{
		"incidents", "vulns", "posture", "risks", "attack-chains",
		"inventory", "network-policies", "seccomp", "cloud-accounts",
		"runtime-rules", "runtime-policies", "integrations", "repo-posture",
	}
	have := map[string]bool{}
	for _, c := range root.Commands() {
		have[c.Name()] = true
	}
	for _, w := range want {
		if !have[w] {
			t.Errorf("missing cluster command %q", w)
		}
	}
}
