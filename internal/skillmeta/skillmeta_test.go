package skillmeta_test

import (
	"testing"

	"github.com/armosec/armoctl/internal/skillmeta"
)

func TestRegisterAndAll(t *testing.T) {
	skillmeta.Reset()
	defer skillmeta.Reset()

	skillmeta.Register(skillmeta.Meta{Name: "armoctl-foo", Cluster: "foo"})
	skillmeta.Register(skillmeta.Meta{Name: "armoctl-bar", Cluster: "bar"})

	all := skillmeta.All()
	if len(all) != 2 {
		t.Fatalf("want 2, got %d", len(all))
	}

	got := skillmeta.ByCluster("bar")
	if got == nil || got.Name != "armoctl-bar" {
		t.Fatalf("ByCluster: %+v", got)
	}
	if skillmeta.ByCluster("missing") != nil {
		t.Fatal("expected nil for unknown cluster")
	}
}

func TestRegisterRejectsDuplicate(t *testing.T) {
	skillmeta.Reset()
	defer skillmeta.Reset()

	skillmeta.Register(skillmeta.Meta{Name: "armoctl-foo", Cluster: "foo"})
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on duplicate register")
		}
	}()
	skillmeta.Register(skillmeta.Meta{Name: "armoctl-foo", Cluster: "foo"})
}
