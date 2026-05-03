// Package skillmeta is the curation surface that drives auto-generated
// per-cluster skill documentation. Each cluster's package init() calls
// Register with its Meta. The gen-skill-docs binary then walks the
// registry and the cobra command tree to produce skills/armoctl-<cluster>/SKILL.md.
package skillmeta

import "fmt"

type Field struct {
	Name string
	Doc  string
}

type Recipe struct {
	Title string
	Body  string // arbitrary markdown
}

type Meta struct {
	// Name is the skill name including the "armoctl-" prefix.
	Name string
	// Cluster is the cobra subcommand name (e.g. "vulns") used to look up
	// the corresponding *cobra.Command when rendering.
	Cluster string
	// Description goes into the SKILL.md frontmatter and drives skill matching.
	Description string
	// Summary is a free-form paragraph rendered at the top of the skill.
	Summary string
	// FieldNotes maps a field name (must exist in Cheatsheet) to a one-or-two
	// sentence semantic explanation that cobra/Cheatsheet cannot provide.
	FieldNotes map[string]string
	// Cheatsheet is a copy of the cluster's cheatsheet, scoped by sub-resource.
	Cheatsheet map[string][]Field
	// Recipes are curated worked examples.
	Recipes []Recipe
}

var registry []Meta

func Register(m Meta) {
	for _, existing := range registry {
		if existing.Cluster == m.Cluster {
			panic(fmt.Sprintf("skillmeta: duplicate registration for cluster %q", m.Cluster))
		}
	}
	registry = append(registry, m)
}

func All() []Meta {
	out := make([]Meta, len(registry))
	copy(out, registry)
	return out
}

func ByCluster(cluster string) *Meta {
	for i := range registry {
		if registry[i].Cluster == cluster {
			m := registry[i]
			return &m
		}
	}
	return nil
}

// Reset clears the registry. Tests only.
func Reset() {
	registry = nil
}
