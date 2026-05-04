// cmd/gen-skill-docs/main.go
//
// gen-skill-docs walks the cobra command tree and the skillmeta registry,
// then writes one skills/armoctl-<cluster>/SKILL.md per registered cluster.
//
// Usage: go run ./cmd/gen-skill-docs [-out skills]
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/armosec/armoctl/internal/rootcmd"
	"github.com/armosec/armoctl/internal/skillmeta"
)

func main() {
	out := flag.String("out", "skills", "output directory")
	flag.Parse()

	root := rootcmd.NewRootCmd()
	// byName maps cobra top-level subcommand name -> command. The skillmeta
	// contract is "Meta.Cluster equals a top-level subcommand name". If
	// clusters ever become hierarchical (e.g. "armoctl cloud aws"), this
	// lookup must change.
	byName := make(map[string]*cobra.Command)
	for _, c := range root.Commands() {
		byName[c.Name()] = c
	}

	all := skillmeta.All()
	sort.Slice(all, func(i, j int) bool { return all[i].Cluster < all[j].Cluster })

	// Reconcile first: a registered cluster with no cobra subcommand is a
	// programmer bug. Fail without writing anything so the working tree
	// never ends up partially regenerated.
	var missing []string
	for _, m := range all {
		if _, ok := byName[m.Cluster]; !ok {
			missing = append(missing, m.Cluster)
		}
	}
	if len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "skillmeta registered clusters not found in cobra tree: %v\n", missing)
		os.Exit(2)
	}

	// All clusters resolve. Render and write.
	for _, m := range all {
		c := byName[m.Cluster]
		body := renderSkill(m, c)
		dir := filepath.Join(*out, m.Name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			die(fmt.Errorf("mkdir %s for cluster %s: %w", dir, m.Cluster, err))
		}
		path := filepath.Join(dir, "SKILL.md")
		if err := os.WriteFile(path, body, 0o644); err != nil {
			die(fmt.Errorf("write %s for cluster %s: %w", path, m.Cluster, err))
		}
		fmt.Printf("wrote %s\n", path)
	}

}

func die(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
