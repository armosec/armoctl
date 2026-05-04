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

func TestNewRootCmdHasNoExtraCommands(t *testing.T) {
	root := rootcmd.NewRootCmd()
	if got := len(root.Commands()); got != 13 {
		names := make([]string, 0, got)
		for _, c := range root.Commands() {
			names = append(names, c.Name())
		}
		t.Errorf("NewRootCmd should have exactly 13 cluster commands, got %d: %v", got, names)
	}
	for _, banned := range []string{"ecs", "configure", "schema", "version", "update"} {
		for _, c := range root.Commands() {
			if c.Name() == banned {
				t.Errorf("%q should NOT be in factory tree (it's a main-only concern)", banned)
			}
		}
	}
}
