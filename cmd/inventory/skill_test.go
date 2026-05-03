package inventory

import (
	"testing"

	"github.com/armosec/armoctl/internal/skillmeta"
)

func TestSkillRegistered(t *testing.T) {
	got := skillmeta.ByCluster("inventory")
	if got == nil {
		t.Fatal("not registered")
	}
	if got.Description == "" {
		t.Error("Description empty")
	}
	if len(got.Cheatsheet) == 0 {
		t.Error("Cheatsheet empty")
	}
}

func TestFieldNotesAreSubsetOfCheatsheet(t *testing.T) {
	m := skillmeta.ByCluster("inventory")
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
