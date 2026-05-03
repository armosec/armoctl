package vulns

import (
	"testing"

	"github.com/armosec/armoctl/internal/skillmeta"
)

func TestSkillRegistered(t *testing.T) {
	got := skillmeta.ByCluster("vulns")
	if got == nil {
		t.Fatal("vulns not registered")
	}
	if got.Name != "armoctl-vulns" {
		t.Errorf("Name=%q", got.Name)
	}
	if got.Description == "" {
		t.Error("Description empty")
	}
	if len(got.Cheatsheet) == 0 {
		t.Error("Cheatsheet empty")
	}
}

func TestFieldNotesAreSubsetOfCheatsheet(t *testing.T) {
	m := skillmeta.ByCluster("vulns")
	known := map[string]bool{}
	for _, fields := range m.Cheatsheet {
		for _, f := range fields {
			known[f.Name] = true
		}
	}
	for name := range m.FieldNotes {
		if !known[name] {
			t.Errorf("FieldNotes references %q which is not in Cheatsheet", name)
		}
	}
}
