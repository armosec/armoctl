package main

import (
	"os"
	"testing"

	"github.com/spf13/cobra"

	"github.com/armosec/armoctl/internal/skillmeta"
)

func fakeFooCmd() *cobra.Command {
	root := &cobra.Command{Use: "foo", Short: "Foo cluster"}
	list := &cobra.Command{Use: "list", Short: "List foos"}
	list.Flags().String("severity", "", "Filter by severity")
	list.Flags().Int("page-size", 50, "Page size")
	get := &cobra.Command{Use: "get <guid>", Short: "Get a foo by GUID"}
	root.AddCommand(list, get)
	return root
}

func fakeFooMeta() skillmeta.Meta {
	return skillmeta.Meta{
		Name:        "armoctl-foo",
		Cluster:     "foo",
		Description: "Description for the foo cluster.",
		Summary:     "Summary paragraph for foo.",
		Cheatsheet: map[string][]skillmeta.Field{
			"foo": {
				{Name: "guid", Doc: "Unique identifier"},
				{Name: "severity", Doc: "Severity level"},
			},
		},
		FieldNotes: map[string]string{
			"severity": "Severity is post-policy, not raw CVSS.",
		},
		Recipes: []skillmeta.Recipe{
			{Title: "List Critical foos", Body: "```\narmoctl foo list --severity Critical\n```"},
		},
	}
}

func TestRenderSkill_Golden(t *testing.T) {
	got := renderSkill(fakeFooMeta(), fakeFooCmd())
	want, err := os.ReadFile("testdata/golden/foo.md")
	if err != nil {
		// To regenerate: WRITE_GOLDEN=1 go test ./cmd/gen-skill-docs/...
		if os.Getenv("WRITE_GOLDEN") != "" {
			_ = os.WriteFile("testdata/golden/foo.md", got, 0o644)
			t.Log("golden created")
			return
		}
		t.Fatal(err)
	}
	if string(got) != string(want) {
		// To regenerate: WRITE_GOLDEN=1 go test ./cmd/gen-skill-docs/...
		if os.Getenv("WRITE_GOLDEN") != "" {
			_ = os.WriteFile("testdata/golden/foo.md", got, 0o644)
			t.Log("golden updated")
			return
		}
		t.Errorf("renderSkill mismatch.\nGot:\n%s\nWant:\n%s", got, want)
	}
}
