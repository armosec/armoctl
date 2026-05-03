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
	byName := make(map[string]*cobra.Command)
	for _, c := range root.Commands() {
		byName[c.Name()] = c
	}

	all := skillmeta.All()
	sort.Slice(all, func(i, j int) bool { return all[i].Cluster < all[j].Cluster })

	missing := []string{}
	for _, m := range all {
		c, ok := byName[m.Cluster]
		if !ok {
			missing = append(missing, m.Cluster)
			continue
		}
		body := renderSkill(m, c)
		dir := filepath.Join(*out, m.Name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			die(err)
		}
		path := filepath.Join(dir, "SKILL.md")
		if err := os.WriteFile(path, body, 0o644); err != nil {
			die(err)
		}
		fmt.Printf("wrote %s\n", path)
	}

	if len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "warning: skillmeta registered clusters not found in cobra tree: %v\n", missing)
		os.Exit(2)
	}
}

func die(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
