# armoctl Claude Plugin Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Promote the existing `SKILL.md` into a full Claude Code plugin (and Gemini extension) shipped from the same repo and release tags as the binary, with auto-generated per-cluster skills and a SessionStart hook that keeps the binary in lockstep with the plugin version.

**Architecture:** A new `internal/skillmeta` package defines a process-wide registry. Each cluster gets a `cmd/<cluster>/skill.go` that calls `skillmeta.Register(...)` from its `init()`. A new `cmd/gen-skill-docs` binary walks the cobra command tree and the registry to emit `skills/armoctl-<cluster>/SKILL.md` per cluster. Manifests for Claude Code and Gemini, plus a SessionStart hook, complete the plugin layout.

**Tech Stack:** Go 1.23+, cobra, viper, bash, JSON manifests. Tests via `go test`. Hook tested with `bats`-style shell tests written in Go (no extra dep).

**Spec:** [`docs/superpowers/specs/2026-05-03-armoctl-claude-plugin-design.md`](../specs/2026-05-03-armoctl-claude-plugin-design.md)

---

## File-level decomposition

| File | Created/Modified | Responsibility |
|---|---|---|
| `internal/skillmeta/skillmeta.go` | Create | Types (`Meta`, `Recipe`, `Field`) + process-wide registry. |
| `internal/skillmeta/skillmeta_test.go` | Create | Registry behaviour. |
| `internal/rootcmd/rootcmd.go` | Create | Factory `NewRootCmd()` so generator and `main` share one tree. |
| `root.go` | Modify | Delegate command-tree construction to `internal/rootcmd`. |
| `cmd/<cluster>/skill.go` × 13 | Create | Per-cluster `init()` registering a `skillmeta.Meta` populated with curated description, summary, field notes, recipes, and a copy of `Cheatsheet()`. |
| `cmd/<cluster>/skill_test.go` × 13 | Create | Assert `FieldNotes` keys are a subset of `Cheatsheet` keys. |
| `cmd/gen-skill-docs/main.go` | Create | Entry point — calls `internal/rootcmd.NewRootCmd()`, walks tree + registry, writes per-cluster skill files. |
| `cmd/gen-skill-docs/render.go` | Create | Pure rendering: `(Meta, *cobra.Command) → []byte`. Easy to test. |
| `cmd/gen-skill-docs/render_test.go` | Create | Golden-file tests with fake `Meta` + fake `cobra.Command`. |
| `cmd/gen-skill-docs/testdata/golden/*.md` | Create | Golden outputs for the renderer tests. |
| `Makefile` | Modify | Add `skill-docs` and `verify-skill-docs` targets. |
| `skills/armoctl/SKILL.md` | Create | Hand-written root skill (ported from current `SKILL.md`, scoped down to setup + contracts + index). |
| `skills/armoctl-<cluster>/SKILL.md` × 13 | Generated | Output of `make skill-docs`. Committed to git. |
| `.claude-plugin/plugin.json` | Create | Claude Code plugin manifest. |
| `.claude-plugin/marketplace.json` | Create | Self-hosted marketplace listing. |
| `gemini-extension.json` | Create | Gemini CLI manifest. |
| `hooks/session-start.sh` | Create | Binary presence + version check + auto-install/update. |
| `hooks/session_start_test.go` | Create | Drives the hook with stubbed `armoctl` binaries via `PATH`. |
| `.github/workflows/skill-docs.yaml` | Create | CI gate: `make verify-skill-docs`. |
| `.github/workflows/pr-merged.yaml` | Modify | Bump `plugin.json` and `marketplace.json` `version` on release. |
| `SKILL.md` (root) | Delete | Superseded by `skills/armoctl/SKILL.md`. |
| `README.md` | Modify | Replace old SKILL.md pointer with plugin install instructions. |

---

### Task 1: skillmeta package — types + registry

**Files:**
- Create: `internal/skillmeta/skillmeta.go`
- Create: `internal/skillmeta/skillmeta_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/skillmeta/skillmeta_test.go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/skillmeta/...`
Expected: FAIL with "no Go files" or "undefined: skillmeta.Meta".

- [ ] **Step 3: Implement skillmeta**

```go
// internal/skillmeta/skillmeta.go
//
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/skillmeta/...`
Expected: PASS, both tests.

- [ ] **Step 5: Commit**

```bash
git add internal/skillmeta/
git commit -m "feat: skillmeta package — registry for per-cluster skill metadata"
```

---

### Task 2: Pilot cluster registration — vulns

This task validates the registry shape with the most interesting cluster (`vulns` has the `inUse` semantic the user explicitly called out). The same pattern applies to the other 12 clusters in Task 7.

**Files:**
- Create: `cmd/vulns/skill.go`
- Create: `cmd/vulns/skill_test.go`

- [ ] **Step 1: Write the failing test**

```go
// cmd/vulns/skill_test.go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/vulns/ -run TestSkill`
Expected: FAIL with "vulns not registered".

- [ ] **Step 3: Implement skill.go**

The implementer should review the existing `cmd/vulns/fields.go` to confirm the Cheatsheet scopes and field names referenced below match (the `inUse` and `fixVersion` field-note keys MUST exist in the Cheatsheet — the test will fail otherwise). Adjust field names to actual ones if needed.

```go
// cmd/vulns/skill.go
package vulns

import "github.com/armosec/armoctl/internal/skillmeta"

func init() {
	skillmeta.Register(skillmeta.Meta{
		Name:    "armoctl-vulns",
		Cluster: "vulns",
		Description: "ARMO vulnerability triage — list CVEs, find affected images/hosts/workloads, " +
			"check in-use status, manage exception policies. Use when the user is investigating " +
			"package vulnerabilities, container CVEs, or remediation prioritization.",
		Summary: "The `vulns` cluster covers the runtime + scan vulnerability surface. The most " +
			"important triage axis is `inUse`: ARMO observes which packages are actually loaded " +
			"in running workloads, so a Critical CVE in dormant code is a much lower priority " +
			"than the same CVE in an in-use library. Always filter by inUse when scoping urgent work.",
		FieldNotes: map[string]string{
			"inUse":      "Runtime-loaded vs. dormant on disk. Critical for triage: a Critical CVE in dormant code is much lower priority than the same CVE in an in-use library. Filter with `--query '.items[] | select(.attributes.inUse == true)'`.",
			"fixVersion": "Empty string means no fix available upstream — don't suggest 'upgrade' as a remediation in that case.",
			"severity":   "ARMO severity, not raw CVSS — already adjusted for runtime context and exception policies.",
		},
		Cheatsheet: convertCheatsheet(Cheatsheet()),
		Recipes: []skillmeta.Recipe{
			{
				Title: "Critical CVEs that are actually in use",
				Body:  "```\narmoctl vulns list --severity Critical --query '.items[] | select(.attributes.inUse == true)'\n```",
			},
			{
				Title: "List exceptions for a CVE",
				Body:  "```\narmoctl vulns exceptions list --cve CVE-2024-12345\n```",
			},
		},
	})
}

func convertCheatsheet(in map[string][]Field) map[string][]skillmeta.Field {
	out := make(map[string][]skillmeta.Field, len(in))
	for k, v := range in {
		fs := make([]skillmeta.Field, len(v))
		for i, f := range v {
			fs[i] = skillmeta.Field{Name: f.Name, Doc: f.Doc}
		}
		out[k] = fs
	}
	return out
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/vulns/ -run TestSkill`
Expected: PASS, both subtests. If `FieldNotes references "inUse" which is not in Cheatsheet" appears, look at `cmd/vulns/fields.go` for the actual field name (it may be `inUseLabel` or capitalised differently) and update `FieldNotes` keys to match. Same for `fixVersion` and `severity`.

- [ ] **Step 5: Commit**

```bash
git add cmd/vulns/skill.go cmd/vulns/skill_test.go
git commit -m "feat(vulns): register skillmeta with curated field semantics"
```

---

### Task 3: Extract NewRootCmd factory

The generator needs the same cobra tree `main()` builds. Today `rootCmd` is package-private inside `package main` — not importable. Move construction into `internal/rootcmd` and have `main` call it.

**Files:**
- Create: `internal/rootcmd/rootcmd.go`
- Modify: `root.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/rootcmd/rootcmd_test.go
package rootcmd_test

import (
	"testing"

	"github.com/armosec/armoctl/internal/rootcmd"
)

func TestNewRootCmdHasAllClusters(t *testing.T) {
	root := rootcmd.NewRootCmd()
	want := []string{
		"incidents", "vulns", "posture", "risks", "attackchains",
		"inventory", "networkpolicies", "seccomp", "cloudaccounts",
		"runtimerules", "runtimepolicies", "integrations", "repoposture",
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/rootcmd/...`
Expected: FAIL — package does not exist.

- [ ] **Step 3: Create the factory**

```go
// internal/rootcmd/rootcmd.go
//
// Package rootcmd builds the cobra command tree. main() and the gen-skill-docs
// generator both call NewRootCmd so they see exactly the same tree.
package rootcmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	attackchainscmd "github.com/armosec/armoctl/cmd/attackchains"
	"github.com/armosec/armoctl/cmd/cliclient"
	"github.com/armosec/armoctl/cmd/cliflags"
	cloudaccountscmd "github.com/armosec/armoctl/cmd/cloudaccounts"
	incidentscmd "github.com/armosec/armoctl/cmd/incidents"
	integrationscmd "github.com/armosec/armoctl/cmd/integrations"
	inventorycmd "github.com/armosec/armoctl/cmd/inventory"
	networkpoliciescmd "github.com/armosec/armoctl/cmd/networkpolicies"
	posturecmd "github.com/armosec/armoctl/cmd/posture"
	repoposturecmd "github.com/armosec/armoctl/cmd/repoposture"
	riskscmd "github.com/armosec/armoctl/cmd/risks"
	runtimepoliciescmd "github.com/armosec/armoctl/cmd/runtimepolicies"
	runtimerulescmd "github.com/armosec/armoctl/cmd/runtimerules"
	seccompcmd "github.com/armosec/armoctl/cmd/seccomp"
	vulnscmd "github.com/armosec/armoctl/cmd/vulns"
)

// NewRootCmd builds and returns the configured root cobra command,
// populated with every cluster subcommand. It does NOT register the
// ECS, configure, schema, or version-check infrastructure that main()
// adds — those concerns belong to main.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "armoctl",
		Short: "ARMO CLI for instrumenting cloud workloads",
		Long:  "armoctl is a CLI tool for instrumenting ECS task definitions with the ARMO runtime security agent.",
	}

	cliflags.Register(root)
	cf := cliclient.Default(viper.GetString)
	root.AddCommand(incidentscmd.Cmd(cf))
	root.AddCommand(vulnscmd.Cmd(cf))
	root.AddCommand(posturecmd.Cmd(cf))
	root.AddCommand(riskscmd.Cmd(cf))
	root.AddCommand(attackchainscmd.Cmd(cf))
	root.AddCommand(inventorycmd.Cmd(cf))
	root.AddCommand(networkpoliciescmd.Cmd(cf))
	root.AddCommand(seccompcmd.Cmd(cf))
	root.AddCommand(cloudaccountscmd.Cmd(cf))
	root.AddCommand(runtimerulescmd.Cmd(cf))
	root.AddCommand(runtimepoliciescmd.Cmd(cf))
	root.AddCommand(integrationscmd.Cmd(cf))
	root.AddCommand(repoposturecmd.Cmd(cf))

	return root
}
```

- [ ] **Step 4: Refactor root.go to use the factory**

Edit `root.go` — replace the bulk of the cluster `AddCommand` block with a call to the factory. Keep the ECS/configure/schema/version-check parts where they are.

```go
// In root.go, replace the existing imports of the cluster packages
// (incidentscmd, vulnscmd, ...) with a single import:
//   "github.com/armosec/armoctl/internal/rootcmd"
// and replace the cliflags.Register + 13 AddCommand calls in init() with:
//
//   built := rootcmd.NewRootCmd()
//   for _, sub := range built.Commands() {
//       rootCmd.AddCommand(sub)
//   }
//   for _, f := range []string{
//       // copy any persistent flag names from cliflags.Register if needed,
//       // OR call cliflags.Register(rootCmd) directly here instead.
//   }
```

Concrete diff for `root.go`:

```diff
 import (
 	"os"
 	"path/filepath"
 	"time"
 
 	"github.com/spf13/cobra"
 	"github.com/spf13/viper"
 
 	ecscmd "github.com/armosec/armoctl/ecs"
-	"github.com/armosec/armoctl/cmd/cliclient"
-	"github.com/armosec/armoctl/cmd/cliflags"
-	attackchainscmd "github.com/armosec/armoctl/cmd/attackchains"
-	cloudaccountscmd "github.com/armosec/armoctl/cmd/cloudaccounts"
-	incidentscmd "github.com/armosec/armoctl/cmd/incidents"
-	integrationscmd "github.com/armosec/armoctl/cmd/integrations"
-	inventorycmd "github.com/armosec/armoctl/cmd/inventory"
-	networkpoliciescmd "github.com/armosec/armoctl/cmd/networkpolicies"
-	posturecmd "github.com/armosec/armoctl/cmd/posture"
-	repoposturecmd "github.com/armosec/armoctl/cmd/repoposture"
-	riskscmd "github.com/armosec/armoctl/cmd/risks"
-	runtimepoliciescmd "github.com/armosec/armoctl/cmd/runtimepolicies"
-	runtimerulescmd "github.com/armosec/armoctl/cmd/runtimerules"
-	seccompcmd "github.com/armosec/armoctl/cmd/seccomp"
-	vulnscmd "github.com/armosec/armoctl/cmd/vulns"
 	"github.com/armosec/armoctl/internal/config"
+	"github.com/armosec/armoctl/internal/rootcmd"
 	schemacmd "github.com/armosec/armoctl/internal/schema"
 	versionpkg "github.com/armosec/armoctl/internal/version"
 )

 func init() {
 	cobra.OnInitialize(initConfig)

 	rootCmd.AddCommand(ecscmd.EcsCmd)
 	rootCmd.AddCommand(configureCmd)

-	cliflags.Register(rootCmd)
-	rootCmd.AddCommand(incidentscmd.Cmd(cliclient.Default(viper.GetString)))
-	rootCmd.AddCommand(vulnscmd.Cmd(cliclient.Default(viper.GetString)))
-	rootCmd.AddCommand(posturecmd.Cmd(cliclient.Default(viper.GetString)))
-	rootCmd.AddCommand(riskscmd.Cmd(cliclient.Default(viper.GetString)))
-	rootCmd.AddCommand(attackchainscmd.Cmd(cliclient.Default(viper.GetString)))
-	rootCmd.AddCommand(inventorycmd.Cmd(cliclient.Default(viper.GetString)))
-	rootCmd.AddCommand(networkpoliciescmd.Cmd(cliclient.Default(viper.GetString)))
-	rootCmd.AddCommand(seccompcmd.Cmd(cliclient.Default(viper.GetString)))
-	rootCmd.AddCommand(cloudaccountscmd.Cmd(cliclient.Default(viper.GetString)))
-	rootCmd.AddCommand(runtimerulescmd.Cmd(cliclient.Default(viper.GetString)))
-	rootCmd.AddCommand(runtimepoliciescmd.Cmd(cliclient.Default(viper.GetString)))
-	rootCmd.AddCommand(integrationscmd.Cmd(cliclient.Default(viper.GetString)))
-	rootCmd.AddCommand(repoposturecmd.Cmd(cliclient.Default(viper.GetString)))
+	built := rootcmd.NewRootCmd()
+	for _, sub := range built.Commands() {
+		rootCmd.AddCommand(sub)
+	}
+	rootCmd.PersistentFlags().AddFlagSet(built.PersistentFlags())
 	rootCmd.AddCommand(schemacmd.Cmd())
```

- [ ] **Step 5: Run all tests**

Run: `go test ./...`
Expected: PASS. The CLI binary still builds (`go build ./...`) and `armoctl --help` shows the same subcommands as before.

- [ ] **Step 6: Smoke test the binary**

Run: `go run . --help`
Expected: Output lists `incidents, vulns, posture, risks, attackchains, …` exactly as before.

- [ ] **Step 7: Commit**

```bash
git add internal/rootcmd/ root.go
git commit -m "refactor: extract NewRootCmd factory shared by main and generators"
```

---

### Task 4: Skill renderer (pure markdown rendering, golden-tested)

Build the renderer in isolation first — no I/O, no real cobra tree needed. Tests use fake `Meta` plus a fake mini cobra tree.

**Files:**
- Create: `cmd/gen-skill-docs/render.go`
- Create: `cmd/gen-skill-docs/render_test.go`
- Create: `cmd/gen-skill-docs/testdata/golden/foo.md`

- [ ] **Step 1: Write the failing test**

```go
// cmd/gen-skill-docs/render_test.go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/gen-skill-docs/...`
Expected: FAIL — `renderSkill` undefined.

- [ ] **Step 3: Implement the renderer**

```go
// cmd/gen-skill-docs/render.go
package main

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/armosec/armoctl/internal/skillmeta"
)

// renderSkill returns the SKILL.md byte content for the given cluster.
// It is a pure function: same inputs → same output, no I/O.
func renderSkill(m skillmeta.Meta, clusterCmd *cobra.Command) []byte {
	var b bytes.Buffer

	// Frontmatter
	fmt.Fprintln(&b, "---")
	fmt.Fprintf(&b, "name: %s\n", m.Name)
	fmt.Fprintf(&b, "description: %s\n", oneLine(m.Description))
	fmt.Fprintln(&b, "---")
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "# %s\n\n", m.Name)
	fmt.Fprintf(&b, "%s\n\n", m.Summary)

	// Commands
	fmt.Fprintln(&b, "## Commands")
	fmt.Fprintln(&b)
	if clusterCmd != nil {
		writeCommandTree(&b, clusterCmd, "")
	}
	fmt.Fprintln(&b)

	// Resource fields
	if len(m.Cheatsheet) > 0 {
		fmt.Fprintln(&b, "## Resource fields")
		fmt.Fprintln(&b)
		scopes := make([]string, 0, len(m.Cheatsheet))
		for s := range m.Cheatsheet {
			scopes = append(scopes, s)
		}
		sort.Strings(scopes)
		for _, s := range scopes {
			fmt.Fprintf(&b, "### %s\n\n", s)
			fmt.Fprintln(&b, "| Field | Description |")
			fmt.Fprintln(&b, "|---|---|")
			for _, f := range m.Cheatsheet[s] {
				fmt.Fprintf(&b, "| `%s` | %s |\n", f.Name, escapePipes(f.Doc))
			}
			fmt.Fprintln(&b)
		}
	}

	// Field semantics
	if len(m.FieldNotes) > 0 {
		fmt.Fprintln(&b, "## Field semantics")
		fmt.Fprintln(&b)
		names := make([]string, 0, len(m.FieldNotes))
		for n := range m.FieldNotes {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			fmt.Fprintf(&b, "**`%s`** — %s\n\n", n, m.FieldNotes[n])
		}
	}

	// Recipes
	if len(m.Recipes) > 0 {
		fmt.Fprintln(&b, "## Recipes")
		fmt.Fprintln(&b)
		for _, r := range m.Recipes {
			fmt.Fprintf(&b, "### %s\n\n", r.Title)
			fmt.Fprintf(&b, "%s\n\n", r.Body)
		}
	}

	return b.Bytes()
}

func writeCommandTree(b *bytes.Buffer, cmd *cobra.Command, prefix string) {
	for _, sub := range cmd.Commands() {
		if sub.Hidden || !sub.IsAvailableCommand() {
			continue
		}
		full := strings.TrimSpace(prefix + " " + sub.Use)
		fmt.Fprintf(b, "- `%s` — %s\n", full, sub.Short)
		if hasFlags(sub) {
			fmt.Fprintln(b, "  - Flags:")
			sub.LocalFlags().VisitAll(func(f *flag) {
				fmt.Fprintf(b, "    - `--%s` (%s) %s\n", f.Name, f.Value.Type(), f.Usage)
			})
		}
		if len(sub.Commands()) > 0 {
			writeCommandTree(b, sub, full)
		}
	}
}

func hasFlags(c *cobra.Command) bool {
	any := false
	c.LocalFlags().VisitAll(func(*flag) { any = true })
	return any
}

func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func escapePipes(s string) string {
	return strings.ReplaceAll(s, "|", `\|`)
}

// flag is an alias to keep the import surface small.
type flag = pflagFlag
```

Add the `flag` alias in a tiny adjacent file because `pflag.Flag` is the actual type:

```go
// cmd/gen-skill-docs/flagalias.go
package main

import "github.com/spf13/pflag"

type pflagFlag = pflag.Flag
```

- [ ] **Step 4: Generate the golden file**

```bash
mkdir -p cmd/gen-skill-docs/testdata/golden
WRITE_GOLDEN=1 go test ./cmd/gen-skill-docs/... -run TestRenderSkill_Golden
```

Expected: PASS, golden file written. Open `cmd/gen-skill-docs/testdata/golden/foo.md` and inspect — it should look like a valid SKILL.md with frontmatter, commands list, fields table, semantics, and one recipe.

- [ ] **Step 5: Re-run without WRITE_GOLDEN to verify deterministic**

Run: `go test ./cmd/gen-skill-docs/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add cmd/gen-skill-docs/
git commit -m "feat: skill renderer with golden tests"
```

---

### Task 5: Generator entry point (real cluster → real file)

**Files:**
- Create: `cmd/gen-skill-docs/main.go`

- [ ] **Step 1: Implement the entry point**

The renderer is already tested. Main is a thin shell: load the registry (population happens via side-effect imports through `internal/rootcmd`), look up each cluster's cobra subcommand, render, write.

```go
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

	"github.com/armosec/armoctl/internal/rootcmd"
	"github.com/armosec/armoctl/internal/skillmeta"
)

func main() {
	out := flag.String("out", "skills", "output directory")
	flag.Parse()

	root := rootcmd.NewRootCmd()
	clusterByName := map[string]*pflagFlagOrCmd{}
	_ = clusterByName // appease linter — replaced below
	subs := root.Commands()
	byName := make(map[string]*cobraCmd, len(subs))
	for _, c := range subs {
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
```

The placeholder type aliases above are noise — replace with real ones:

```go
// in cmd/gen-skill-docs/main.go remove the clusterByName/pflagFlagOrCmd noise
// and rename "cobraCmd" to "cobra.Command" with an import.
```

The cleaner final version:

```go
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
```

- [ ] **Step 2: Run the generator on the pilot (vulns)**

```bash
go run ./cmd/gen-skill-docs
```

Expected stdout: `wrote skills/armoctl-vulns/SKILL.md`. The file exists with frontmatter, commands list, fields table, field semantics, recipes.

- [ ] **Step 3: Inspect output**

```bash
cat skills/armoctl-vulns/SKILL.md | head -50
```

Sanity check: frontmatter present, command list reflects actual `armoctl vulns` subcommands, table contains real field names from `cmd/vulns/fields.go`.

- [ ] **Step 4: Commit**

```bash
git add cmd/gen-skill-docs/main.go skills/armoctl-vulns/
git commit -m "feat(gen-skill-docs): generator entry point + first generated skill (vulns)"
```

---

### Task 6: Makefile targets

**Files:**
- Modify: `Makefile`

- [ ] **Step 1: Add the targets**

```makefile
.PHONY: skill-docs verify-skill-docs

# Regenerate per-cluster skill markdown files.
skill-docs:
	go run ./cmd/gen-skill-docs

# CI gate — fail if generated skills are stale relative to source.
verify-skill-docs:
	@$(MAKE) skill-docs
	@if ! git diff --quiet -- skills/; then \
		echo "ERROR: skills/ is stale. Run 'make skill-docs' and commit the result."; \
		git --no-pager diff -- skills/; \
		exit 1; \
	fi
```

- [ ] **Step 2: Verify both targets**

```bash
make skill-docs
make verify-skill-docs
```

Expected: `make skill-docs` re-emits the same content (no diff). `make verify-skill-docs` succeeds with no output.

- [ ] **Step 3: Commit**

```bash
git add Makefile
git commit -m "build: add skill-docs and verify-skill-docs Make targets"
```

---

### Task 7: Register skillmeta for the remaining 12 clusters

For each cluster below, create `cmd/<cluster>/skill.go` and `cmd/<cluster>/skill_test.go` with the same shape as `cmd/vulns/skill.go` (Task 2). Per-cluster curated content is given inline. Each cluster gets its own commit so review and revert stay granular.

**Pattern (apply for every cluster):**

```go
// cmd/<cluster>/skill.go
package <cluster>

import "github.com/armosec/armoctl/internal/skillmeta"

func init() {
	skillmeta.Register(skillmeta.Meta{
		Name:        "armoctl-<cluster-skill-suffix>",
		Cluster:     "<cobra-cluster-name>",
		Description: "<one-line description, drives skill matching>",
		Summary:     "<paragraph>",
		FieldNotes:  map[string]string{ /* per-cluster */ },
		Cheatsheet:  convertCheatsheet(Cheatsheet()),
		Recipes:     []skillmeta.Recipe{ /* per-cluster */ },
	})
}

func convertCheatsheet(in map[string][]Field) map[string][]skillmeta.Field {
	out := make(map[string][]skillmeta.Field, len(in))
	for k, v := range in {
		fs := make([]skillmeta.Field, len(v))
		for i, f := range v {
			fs[i] = skillmeta.Field{Name: f.Name, Doc: f.Doc}
		}
		out[k] = fs
	}
	return out
}
```

**Test pattern (every cluster):**

```go
// cmd/<cluster>/skill_test.go
package <cluster>

import (
	"testing"

	"github.com/armosec/armoctl/internal/skillmeta"
)

func TestSkillRegistered(t *testing.T) {
	got := skillmeta.ByCluster("<cobra-cluster-name>")
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
	m := skillmeta.ByCluster("<cobra-cluster-name>")
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
```

**Per-cluster content** (Description / Summary / FieldNotes / Recipes). For each cluster, the implementer should:

1. Read the cluster's existing `fields.go` to confirm field names that are referenced in `FieldNotes` actually exist in `Cheatsheet()`. Adjust if needed.
2. Read the cluster's existing commands to make `Recipes` factually correct.
3. Run `go test ./cmd/<cluster>/` after writing — the subset test will catch drift.

#### 7.1 — `incidents`

- **Name:** `armoctl-incidents`
- **Description:** `"ARMO runtime incidents — list active threats, fetch alerts on a single incident, explain an incident's signal, resolve/silence incidents. Use when investigating live runtime alerts or post-mortems."`
- **Summary:** `"The incidents cluster is the live runtime-threat surface. An incident is the unit of triage; it bundles many alerts produced from runtime detection rules. Severity is ARMO-policy-adjusted, not raw alert severity. Use 'incidents alerts <guid>' to get the full alert payload behind an incident before resolving it."`
- **FieldNotes:**
  - `severity`: `"ARMO-policy severity. Already factored in suppression, false-positive marks, and rule confidence."`
  - `incidentStatus`: `"Live state machine: open → acknowledged → resolved/silenced. 'silenced' means the same signature won't page again for the configured cooldown."`
- **Recipes:**
  - `"List Critical open incidents"` → `"```\narmoctl incidents list --severity Critical --status open\n```"`
  - `"Get all alerts for an incident"` → `"```\narmoctl incidents alerts <incident-guid>\n```"`

- [ ] **Step 7.1.1:** Write `cmd/incidents/skill.go` + `cmd/incidents/skill_test.go` using the patterns above and the content here.
- [ ] **Step 7.1.2:** Run `go test ./cmd/incidents/`. Expected: PASS.
- [ ] **Step 7.1.3:** Commit: `git add cmd/incidents/skill*.go && git commit -m "feat(incidents): register skillmeta"`.

#### 7.2 — `posture`

- **Description:** `"Kubernetes posture scanning — controls, frameworks, exceptions. Use when assessing compliance posture (NSA, MITRE, etc.) or managing posture exception policies."`
- **Summary:** `"Posture is config-time scanning of K8s resources against control frameworks. A 'failed control' means a resource violates a rule from a framework like NSA-CISA. Exception policies suppress specific (control × resource) pairs."`
- **FieldNotes:**
  - `controlID`: `"Stable identifier across framework versions (e.g., C-0001). Prefer this over name when scripting."`
  - `frameworkName`: `"Multi-framework: a single control can belong to several frameworks (NSA, MITRE, ArmoBest, etc.)."`
- **Recipes:**
  - `"List failed controls in a cluster"` → `"```\narmoctl posture controls --cluster <name> --status failed\n```"`

- [ ] **Step 7.2.1:** Files + tests.
- [ ] **Step 7.2.2:** Run `go test ./cmd/posture/`. Expected: PASS.
- [ ] **Step 7.2.3:** Commit: `feat(posture): register skillmeta`.

#### 7.3 — `risks`

- **Description:** `"Security risks (cross-cutting risk view) — list/resources/severities and exception policies. Use when working with the unified ARMO risk score, not per-domain CVE/posture findings."`
- **Summary:** `"Risks are the unified prioritisation surface that combines vulnerability + posture + runtime signal into a single severity per (resource × risk-class). Exception policies live here too."`
- **FieldNotes:**
  - `severity`: `"Composite severity — already accounts for runtime context, exceptions, and exposure."`
  - `policyIDs`: `"On exceptions: the risk IDs the exception applies to. Single-element in current API even though it's an array."`
- **Recipes:**
  - `"List Critical risks"` → `"```\narmoctl risks list --severity Critical\n```"`
  - `"Create an exception with a 30-day expiry"` → `"```\narmoctl risks exceptions create --risk-id <id> --reason 'planned remediation' --expires 30d --dry-run\n```"`

- [ ] **Step 7.3.1:** Files + tests.
- [ ] **Step 7.3.2:** Run `go test ./cmd/risks/`. Expected: PASS.
- [ ] **Step 7.3.3:** Commit.

#### 7.4 — `attackchains`

- **Description:** `"Attack chains — multi-step kill-chain views built by ARMO from runtime + posture signal. Use when the user wants to understand how vulnerabilities chain into reachable exploit paths."`
- **Summary:** `"An attack chain links a posture weakness, a vulnerable component, and runtime context into a sequence an attacker could traverse. List view shows the highest-severity chains; details show the per-step evidence."`
- **FieldNotes:**
  - `chainStatus`: `"open / mitigated / accepted. Acceptance flows through the risks-exceptions cluster, not here."`
- **Recipes:**
  - `"List active attack chains"` → `"```\narmoctl attackchains list --status open\n```"`

- [ ] Steps 7.4.1–3.

#### 7.5 — `inventory`

- **Description:** `"Cluster inventory — list workloads, get unique values for a field. Use to enumerate or pivot on resources before applying another command."`
- **Summary:** `"Inventory is the index of everything ARMO has seen. Use 'inventory list' to enumerate workloads/resources and 'inventory unique-values' to discover the legal values for a given field (clusters, namespaces, kinds, etc.)."`
- **FieldNotes:**
  - `kind`: `"K8s kind — Deployment, StatefulSet, DaemonSet, Job, CronJob, Pod. Use 'inventory unique-values --field kind' to confirm the spelling expected by other commands."`
- **Recipes:**
  - `"List unique namespaces in a cluster"` → `"```\narmoctl inventory unique-values --field namespace --cluster <name>\n```"`

- [ ] Steps 7.5.1–3.

#### 7.6 — `networkpolicies`

- **Description:** `"Generated NetworkPolicies — list discovered policies and generate one for a workload from observed traffic. Use to harden cluster network egress/ingress."`
- **Summary:** `"ARMO observes runtime traffic and emits a least-privilege NetworkPolicy YAML for any selected workload. List shows historical policies; generate produces one on-demand."`
- **FieldNotes:**
  - `workloadKind`: `"Must match an actual workload kind in the target cluster (Deployment, StatefulSet, etc.). Use 'inventory unique-values' to verify."`
- **Recipes:**
  - `"Generate a policy for a workload"` → `"```\narmoctl networkpolicies generate --cluster <c> --namespace <ns> --workload <name>\n```"`

- [ ] Steps 7.6.1–3.

#### 7.7 — `seccomp`

- **Description:** `"Generated seccomp profiles — list and generate profiles per workload. Use to restrict syscalls to those observed at runtime."`
- **Summary:** `"Same model as networkpolicies but for seccomp: ARMO records the syscall set at runtime and emits a tight allow-list profile."`
- **FieldNotes:**
  - `profileScope`: `"Workload-level vs container-level. Container-level is more precise but harder to operate."`
- **Recipes:**
  - `"Generate a profile for a workload"` → `"```\narmoctl seccomp generate --cluster <c> --namespace <ns> --workload <name>\n```"`

- [ ] Steps 7.7.1–3.

#### 7.8 — `cloudaccounts`

- **Description:** `"Cloud account onboarding — list/connect/disconnect ECS accounts. Use to see which AWS accounts ARMO is monitoring or to onboard a new one."`
- **Summary:** `"Cloud accounts is the AWS-side onboarding surface. Today it covers ECS account connection state; future cloud surfaces (EKS, GCP) will land here."`
- **FieldNotes:**
  - `connectionStatus`: `"connected / pending / failed. 'pending' means CloudFormation rollout is still in progress."`
- **Recipes:**
  - `"List ECS accounts"` → `"```\narmoctl cloudaccounts ecs list\n```"`

- [ ] Steps 7.8.1–3.

#### 7.9 — `runtimerules`

- **Description:** `"Runtime detection rules — CRUD on the per-rule policy surface (the ARMO equivalent of a Falco rule). Use to add, modify, or evaluate runtime rules."`
- **Summary:** `"A rule is the smallest unit of runtime detection: 'fire when X happens.' Rules are bundled into runtime policies (next cluster). 'evaluate' lets you check whether a hypothetical event would have fired a given rule."`
- **FieldNotes:**
  - `severity`: `"Rule's contribution to the incident severity, not the alert severity."`
- **Recipes:**
  - `"Create a rule from JSON"` → `"```\narmoctl runtimerules create --file rule.json --dry-run\n```"`

- [ ] Steps 7.9.1–3.

#### 7.10 — `runtimepolicies`

- **Description:** `"Runtime policies — bundles of rules attached to clusters/namespaces/workloads. Use to manage which detection rules apply where."`
- **Summary:** `"A policy is a bag of runtimerules with a binding scope (cluster, namespace, workload). When a workload runs, the union of policies that bind to it determines which rules evaluate."`
- **FieldNotes:**
  - `bindingScope`: `"cluster / namespace / workload. Most-specific binding wins on conflict."`
- **Recipes:**
  - `"List policies bound to a cluster"` → `"```\narmoctl runtimepolicies list --cluster <name>\n```"`

- [ ] Steps 7.10.1–3.

#### 7.11 — `integrations`

- **Description:** `"Outbound integrations — alert channels (Slack/email/webhook), SIEM forwarders, Jira ticket creation. Use to wire ARMO into external workflows."`
- **Summary:** `"Integrations is where ARMO emits, not consumes. Alert channels deliver events; SIEM forwarders ship logs; Jira lets the agent open tickets directly."`
- **FieldNotes:**
  - `channelType`: `"slack / email / webhook / pagerduty / msteams. Each type has its own auth flow."`
- **Recipes:**
  - `"Create a Jira ticket from an incident"` → `"```\narmoctl integrations jira create-ticket --incident <guid> --project <key> --issuetype Bug\n```"`

- [ ] Steps 7.11.1–3.

#### 7.12 — `repoposture`

- **Description:** `"Repository posture — IaC scanning of a connected git repo for config issues, with per-file and per-control views. Use when reviewing posture findings tied to a repo, not a live cluster."`
- **Summary:** `"Same control surface as cluster posture, but the resources are files in a connected git repo. Findings carry both file path and control ID, so they can be deep-linked back to the IaC source."`
- **FieldNotes:**
  - `filePath`: `"Repo-relative. Pair with the repo's commit SHA to deep-link to the exact line."`
- **Recipes:**
  - `"List failed controls in a repo"` → `"```\narmoctl repoposture failed-controls --repo <name>\n```"`

- [ ] Steps 7.12.1–3.

After all 12 sub-clusters are committed:

- [ ] **Step 7.13:** Run `go test ./cmd/...`. Expected: all per-cluster `TestSkillRegistered` and `TestFieldNotesAreSubsetOfCheatsheet` pass.

---

### Task 8: Generate all per-cluster skills

**Files:**
- Create (generated): `skills/armoctl-<cluster>/SKILL.md` × 12 remaining

- [ ] **Step 1: Run the generator**

```bash
make skill-docs
```

Expected stdout: `wrote skills/armoctl-attackchains/SKILL.md`, `wrote skills/armoctl-cloudaccounts/SKILL.md`, … one line per registered cluster (13 total).

- [ ] **Step 2: Inspect a sample**

```bash
cat skills/armoctl-incidents/SKILL.md | head -30
```

Sanity check: frontmatter, summary, real subcommands, real fields.

- [ ] **Step 3: Verify regeneration is deterministic**

```bash
make skill-docs
git diff -- skills/
```

Expected: empty diff. (If not, the renderer has a non-deterministic order somewhere — fix the renderer, not the test.)

- [ ] **Step 4: Run verify target**

```bash
make verify-skill-docs
```

Expected: success, no output.

- [ ] **Step 5: Commit**

```bash
git add skills/
git commit -m "docs(skills): generate per-cluster SKILL.md files"
```

---

### Task 9: Hand-write root skill `skills/armoctl/SKILL.md`

The root skill is **always loaded** when armoctl is in scope. It contains setup, the JSON output contract, the safety contract, and a one-line index pointing at each cluster skill (the description-matcher does the actual routing; the index is for humans). Port content from the existing repo-root `SKILL.md`.

**Files:**
- Create: `skills/armoctl/SKILL.md`

- [ ] **Step 1: Read the current SKILL.md to understand what to keep**

```bash
cat SKILL.md
```

Identify (a) sections that belong in the root skill (setup, output contract, safety contract, error model) and (b) sections that should NOT be in the root skill (per-cluster recipes — those live in `cmd/<cluster>/skill.go` Recipes already).

- [ ] **Step 2: Write the new root skill**

Create `skills/armoctl/SKILL.md` with this structure:

```markdown
---
name: armoctl
description: ARMO security platform CLI — JSON-first agent-friendly access to runtime incidents, vulnerabilities, posture, risks, attack chains, inventory, network policies, seccomp, runtime rules/policies, integrations, cloud accounts, and repository posture. Mutation safety with --dry-run/--yes.
---

# armoctl — ARMO Security Platform CLI

You are a security analyst with `armoctl`. It exposes 13 resource clusters as `armoctl <cluster> <subcommand>`, returns JSON by default, and wraps every mutation with a dry-run/--yes safety contract plus an audit log.

## 1. Setup

[port from current SKILL.md §1]

## 2. Output contract

[port from current SKILL.md §2 — list/get/mutation shapes, --full / --fields / --query]

## 3. Safety contract

[port from current SKILL.md — --dry-run / --yes / TTY confirmation, audit log location, error model with RequestID]

## 4. Cluster index

For cluster-specific commands and field semantics, consult the matching skill:

| Cluster | Skill |
|---|---|
| Runtime incidents | `armoctl-incidents` |
| Vulnerabilities | `armoctl-vulns` |
| Posture | `armoctl-posture` |
| Risks (cross-cutting) | `armoctl-risks` |
| Attack chains | `armoctl-attackchains` |
| Inventory | `armoctl-inventory` |
| Network policies | `armoctl-networkpolicies` |
| Seccomp profiles | `armoctl-seccomp` |
| Runtime rules | `armoctl-runtimerules` |
| Runtime policies | `armoctl-runtimepolicies` |
| Integrations | `armoctl-integrations` |
| Cloud accounts | `armoctl-cloudaccounts` |
| Repository posture | `armoctl-repoposture` |

These skills are auto-loaded by description match when the user's task touches the cluster.
```

(The implementer fills the bracketed sections by porting from the existing repo-root `SKILL.md`. Do not invent content — every fact in the new file must be backed by a fact in the old one.)

- [ ] **Step 3: Verify**

```bash
test -f skills/armoctl/SKILL.md
wc -l skills/armoctl/SKILL.md   # should be ~80–120 lines
```

- [ ] **Step 4: Commit**

```bash
git add skills/armoctl/SKILL.md
git commit -m "docs(skills): root armoctl skill (setup, output contract, safety, cluster index)"
```

---

### Task 10: Plugin manifests

**Files:**
- Create: `.claude-plugin/plugin.json`
- Create: `.claude-plugin/marketplace.json`
- Create: `gemini-extension.json`

- [ ] **Step 1: Determine the current version**

```bash
git describe --tags --abbrev=0
```

Expected output: e.g. `v0.0.6`. Use `0.0.7` (next patch) as the initial plugin version, since the next release that includes this work will tag `v0.0.7`.

- [ ] **Step 2: Create plugin.json**

```bash
mkdir -p .claude-plugin
```

```json
// .claude-plugin/plugin.json
{
  "name": "armoctl",
  "version": "0.0.7",
  "description": "ARMO security platform CLI as a Claude skill — incidents, vulnerabilities, posture, risks, attack chains, runtime, network, integrations.",
  "author": {
    "name": "ARMO",
    "email": "support@armosec.io",
    "url": "https://www.armosec.io"
  },
  "homepage": "https://github.com/armosec/armoctl",
  "repository": "https://github.com/armosec/armoctl",
  "license": "Apache-2.0",
  "keywords": ["security", "kubernetes", "cli", "ecs", "armo", "vulnerabilities", "posture"],
  "skills": "./skills/",
  "hooks": "./hooks/"
}
```

- [ ] **Step 3: Create marketplace.json**

```json
// .claude-plugin/marketplace.json
{
  "name": "armosec",
  "description": "ARMO official Claude plugin marketplace",
  "owner": {
    "name": "ARMO",
    "email": "support@armosec.io"
  },
  "plugins": [
    {
      "name": "armoctl",
      "description": "ARMO security platform CLI as a Claude skill",
      "version": "0.0.7",
      "source": "./",
      "author": {
        "name": "ARMO",
        "email": "support@armosec.io"
      }
    }
  ]
}
```

- [ ] **Step 4: Create gemini-extension.json**

```json
// gemini-extension.json
{
  "name": "armoctl",
  "version": "0.0.7",
  "description": "ARMO security platform CLI as a Gemini extension.",
  "skills": "./skills/"
}
```

(If the Gemini CLI manifest format has evolved, the implementer should consult the Gemini docs at implementation time and adjust — see spec "Open questions". The shape above mirrors the mongodb plugin.)

- [ ] **Step 5: Validate JSON**

```bash
for f in .claude-plugin/plugin.json .claude-plugin/marketplace.json gemini-extension.json; do
  jq . "$f" >/dev/null && echo "$f OK" || { echo "$f BROKEN"; exit 1; }
done
```

Expected: three "OK" lines.

- [ ] **Step 6: Commit**

```bash
git add .claude-plugin/ gemini-extension.json
git commit -m "feat: plugin manifests (Claude Code, marketplace, Gemini)"
```

---

### Task 11: SessionStart hook

**Files:**
- Create: `hooks/session-start.sh`
- Create: `hooks/session_start_test.go`

- [ ] **Step 1: Write the failing test**

```go
// hooks/session_start_test.go
package hooks_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runHook invokes session-start.sh with PATH set so the stub binaries are found.
// Returns combined stdout/stderr.
func runHook(t *testing.T, stubPath, pluginRoot string) (string, error) {
	t.Helper()
	repoRoot, _ := filepath.Abs("..")
	cmd := exec.Command("bash", filepath.Join(repoRoot, "hooks/session-start.sh"))
	cmd.Env = append(os.Environ(),
		"PATH="+stubPath+":"+os.Getenv("PATH"),
		"CLAUDE_PLUGIN_ROOT="+pluginRoot,
	)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// makeStubArmoctl writes a fake armoctl that prints `armoctl <version>`
// when called with --version. If version is empty, the stub is omitted
// (simulating "binary not on PATH").
func makeStubArmoctl(t *testing.T, dir, version string) {
	t.Helper()
	if version == "" {
		return
	}
	script := "#!/usr/bin/env bash\nif [ \"$1\" = \"--version\" ]; then echo armoctl " + version + "; exit 0; fi\nif [ \"$1\" = \"update\" ]; then echo updated > " + filepath.Join(dir, "update_called") + "; exit 0; fi\nexit 0\n"
	path := filepath.Join(dir, "armoctl")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}

func makePluginJSON(t *testing.T, dir, version string) string {
	t.Helper()
	pluginDir := filepath.Join(dir, ".claude-plugin")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"name":"armoctl","version":"` + version + `"}`
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestHook_VersionMatch_NoOp(t *testing.T) {
	stubDir := t.TempDir()
	pluginRoot := makePluginJSON(t, t.TempDir(), "0.0.7")
	makeStubArmoctl(t, stubDir, "0.0.7")

	out, err := runHook(t, stubDir, pluginRoot)
	if err != nil {
		t.Fatalf("hook failed: %v\n%s", err, out)
	}
	if _, statErr := os.Stat(filepath.Join(stubDir, "update_called")); statErr == nil {
		t.Errorf("update should NOT have been called when versions match")
	}
}

func TestHook_VersionMismatch_RunsUpdate(t *testing.T) {
	stubDir := t.TempDir()
	pluginRoot := makePluginJSON(t, t.TempDir(), "0.0.8")
	makeStubArmoctl(t, stubDir, "0.0.7")

	out, err := runHook(t, stubDir, pluginRoot)
	if err != nil {
		t.Fatalf("hook failed: %v\n%s", err, out)
	}
	if _, statErr := os.Stat(filepath.Join(stubDir, "update_called")); statErr != nil {
		t.Errorf("update should have been called when versions differ. Output: %s", out)
	}
	if !strings.Contains(out, "differs from plugin") {
		t.Errorf("expected mismatch message, got: %s", out)
	}
}

func TestHook_BinaryMissing_PrintsInstallHint(t *testing.T) {
	stubDir := t.TempDir() // no armoctl stub
	pluginRoot := makePluginJSON(t, t.TempDir(), "0.0.7")

	out, err := runHook(t, stubDir, pluginRoot)
	// Hook may exit 0 (graceful) even when curl install fails. We only
	// assert it doesn't blow up the session and prints something useful.
	if err != nil && !strings.Contains(out, "armoctl install failed") && !strings.Contains(out, "installing v0.0.7") {
		t.Fatalf("hook failed unexpectedly: %v\n%s", err, out)
	}
	if !strings.Contains(out, "armoctl") {
		t.Errorf("expected armoctl-related output, got: %s", out)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./hooks/...`
Expected: FAIL — `hooks/session-start.sh` does not exist.

- [ ] **Step 3: Write the hook**

```bash
mkdir -p hooks
```

```bash
#!/usr/bin/env bash
# hooks/session-start.sh
#
# SessionStart hook for the armoctl Claude plugin.
# - Ensures the armoctl binary is installed.
# - If installed, ensures it matches the plugin's pinned version.
# - Never blocks session start: prints actionable output and exits 0
#   on failure paths.

set -e

PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"
PLUGIN_VERSION="$(jq -r .version "$PLUGIN_ROOT/.claude-plugin/plugin.json")"

INSTALLED=""
if command -v armoctl >/dev/null 2>&1; then
    INSTALLED="$(armoctl --version 2>/dev/null | awk '{print $NF}')"
fi

# Strip leading 'v' if present for comparison.
norm() { echo "${1#v}"; }

INSTALL_URL="https://package-distribution.armosec.io/armoctl/install.sh"
if [ -z "$INSTALLED" ]; then
    echo "armoctl not found — installing v${PLUGIN_VERSION}…" >&2
    if ! curl -fsSL "$INSTALL_URL" | bash; then
        echo "armoctl install failed; the armoctl skill will not work this session." >&2
        echo "Install manually: curl -fsSL $INSTALL_URL | bash" >&2
        exit 0
    fi
elif [ "$(norm "$INSTALLED")" != "$(norm "$PLUGIN_VERSION")" ]; then
    echo "armoctl ${INSTALLED} differs from plugin ${PLUGIN_VERSION} — running 'armoctl update'…" >&2
    armoctl update || echo "armoctl update failed; continuing with ${INSTALLED}." >&2
fi

exit 0
```

- [ ] **Step 4: Make executable**

```bash
chmod +x hooks/session-start.sh
```

- [ ] **Step 5: Run test**

Run: `go test ./hooks/...`
Expected: PASS, all three subtests.

- [ ] **Step 6: Commit**

```bash
git add hooks/
git commit -m "feat: SessionStart hook for binary install/update"
```

---

### Task 12: CI gate for skill-doc freshness

**Files:**
- Create: `.github/workflows/skill-docs.yaml`

- [ ] **Step 1: Write the workflow**

```yaml
# .github/workflows/skill-docs.yaml
name: skill-docs

on:
  pull_request:
  push:
    branches: [main]

jobs:
  verify:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - name: Verify generated skill docs are up to date
        run: make verify-skill-docs
```

- [ ] **Step 2: Verify locally**

```bash
make verify-skill-docs
```

Expected: success.

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/skill-docs.yaml
git commit -m "ci: gate PRs on regenerated skill docs being committed"
```

---

### Task 13: Release workflow — bump plugin.json on release

**Files:**
- Modify: `.github/workflows/pr-merged.yaml`

- [ ] **Step 1: Read the current workflow**

```bash
cat .github/workflows/pr-merged.yaml
```

Identify the step that computes the new version (e.g., a step that outputs `new_version` like `0.0.8`) and the tag-creation step.

- [ ] **Step 2: Add a version-bump step before tagging**

Insert a step that rewrites the `version` field in both `plugin.json` and `marketplace.json` to the new version, then commits the change to the same release commit. The exact YAML depends on the existing workflow shape — the implementer must read it first.

Pattern (adapt to actual step IDs):

```yaml
      - name: Bump plugin manifests to ${{ steps.bump.outputs.new_version }}
        run: |
          NEW="${{ steps.bump.outputs.new_version }}"
          jq --arg v "$NEW" '.version = $v' .claude-plugin/plugin.json > .claude-plugin/plugin.json.tmp
          mv .claude-plugin/plugin.json.tmp .claude-plugin/plugin.json
          jq --arg v "$NEW" '(.plugins[] | select(.name=="armoctl") | .version) = $v' .claude-plugin/marketplace.json > .claude-plugin/marketplace.json.tmp
          mv .claude-plugin/marketplace.json.tmp .claude-plugin/marketplace.json
          jq --arg v "$NEW" '.version = $v' gemini-extension.json > gemini-extension.json.tmp
          mv gemini-extension.json.tmp gemini-extension.json
          git config user.name "armo-release-bot"
          git config user.email "release@armosec.io"
          git add .claude-plugin/plugin.json .claude-plugin/marketplace.json gemini-extension.json
          git commit -m "chore: bump plugin manifests to v$NEW"
          git push
```

This step must run **before** the tag is pushed so the tag includes the bumped manifests.

- [ ] **Step 3: Add a verify step at the start of the release flow**

Insert a step that runs `make verify-skill-docs` before any version bump or tag — so a release with stale skill docs fails fast:

```yaml
      - name: Verify generated skill docs are fresh
        run: make verify-skill-docs
```

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/pr-merged.yaml
git commit -m "ci(release): bump plugin manifests on release; verify skill docs freshness"
```

---

### Task 14: Retire root SKILL.md and write the README plugin section

The README today documents only the ECS patcher use case. With the plugin landing, it must also explain (a) that armoctl exposes a full agent-driven CLI surface for the ARMO security platform, and (b) how to install the plugin so non-experts can pick it up from Claude Code or Gemini in one step.

**Files:**
- Delete: `SKILL.md`
- Modify: `README.md`

- [ ] **Step 1: Remove the old SKILL.md**

```bash
git rm SKILL.md
```

- [ ] **Step 2: Add a "Use from Claude Code" section near the top of the README**

Open `README.md`. Insert a new section immediately after the one-line description (i.e., after line 3 `CLI tool for instrumenting ECS task definitions...`) and before the existing `## 📦 Install` section. This places the plugin pitch above the manual install so casual readers see it first.

The new section must contain exactly the following content (copy verbatim — the install command line and the plugin name strings are what users will paste):

```markdown
## 🤖 Use from Claude Code or Gemini CLI

armoctl ships as a Claude Code plugin (and Gemini CLI extension) so AI assistants can drive the ARMO security platform directly: list incidents, triage CVEs, manage exception policies, generate network policies, and more.

### Claude Code

```
/plugin marketplace add armosec/armoctl
/plugin install armoctl@armosec
```

The first time a session starts, the plugin checks for the `armoctl` binary on `PATH` and runs the official installer if it's missing. After that, the SessionStart hook keeps the binary on the same version as the plugin (running `armoctl update` whenever they drift).

### Gemini CLI

Add this repo as an extension. The Gemini extension loads the same skills as the Claude plugin from `skills/`. Install the binary first (see the next section).

### What's in the plugin

- A root `armoctl` skill covering setup, the JSON output contract (`--full` / `--fields` / `--query`), the mutation safety contract (`--dry-run` / `--yes`), and error semantics.
- 13 per-cluster skills (`armoctl-incidents`, `armoctl-vulns`, `armoctl-posture`, `armoctl-risks`, `armoctl-attackchains`, `armoctl-inventory`, `armoctl-networkpolicies`, `armoctl-seccomp`, `armoctl-runtimerules`, `armoctl-runtimepolicies`, `armoctl-integrations`, `armoctl-cloudaccounts`, `armoctl-repoposture`) auto-loaded by description match when the user's task touches that cluster.
- A SessionStart hook that ensures the binary is present and version-matched.

### Configure once

```bash
armoctl configure   # interactive — saves to ~/.armoctl/config.yaml
# or via env vars (preferred for headless agents):
export ARMO_CUSTOMER_GUID=...
export ARMO_ACCESS_KEY=...
```

Once configured, the agent can run any read-only command directly. Mutations require `--dry-run` for the preview and `--yes` to commit (or a confirmation prompt on a TTY).
```

- [ ] **Step 3: Verify**

```bash
grep -n "SKILL.md" README.md          # should match nothing
grep -n "marketplace add armosec/armoctl" README.md   # should match in the new section
test ! -f SKILL.md
head -50 README.md                    # confirm new section sits above '## 📦 Install'
```

Expected: no SKILL.md references; the marketplace install command appears; the new "🤖 Use from Claude Code or Gemini CLI" section sits between the one-line description and the existing manual install section.

- [ ] **Step 4: Commit**

```bash
git add README.md
git commit -m "docs(README): add plugin install section, retire root SKILL.md"
```

---

### Task 15: End-to-end smoke

**Files:**
- None modified

- [ ] **Step 1: Full test run**

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 2: Build the binary**

```bash
make armoctl
./armoctl --help
```

Expected: same cluster list as before the refactor.

- [ ] **Step 3: Run the generator end-to-end**

```bash
make verify-skill-docs
```

Expected: success, no diff.

- [ ] **Step 4: Validate every plugin file**

```bash
for f in .claude-plugin/plugin.json .claude-plugin/marketplace.json gemini-extension.json; do
  jq . "$f" >/dev/null && echo "$f OK"
done
ls skills/
```

Expected: 3 OK lines; `skills/` lists `armoctl/` plus 13 `armoctl-<cluster>/` directories.

- [ ] **Step 5: Hook smoke**

```bash
CLAUDE_PLUGIN_ROOT=$(pwd) bash hooks/session-start.sh
```

Expected: either no output (binary already at the pinned version) or a "differs from plugin" message followed by a benign update attempt. Should exit 0.

- [ ] **Step 6: Final commit (if anything was tweaked above)**

If any step revealed a fix, commit it. Otherwise this task is just verification.

---

## Self-review

**Spec coverage check:**

| Spec section | Plan task |
|---|---|
| Repo layout | Task 10 (manifests), Task 11 (hook), Task 9 (root skill), Task 8 (generated skills) |
| `.claude-plugin/plugin.json` | Task 10 |
| `.claude-plugin/marketplace.json` | Task 10 |
| `gemini-extension.json` | Task 10 |
| Root skill | Task 9 |
| Per-cluster skills (auto-gen) | Tasks 4–8 |
| Curation surface (`cmd/<cluster>/skill.go`) | Tasks 2, 7 |
| SessionStart hook | Task 11 |
| Generator | Tasks 4, 5 |
| Make targets | Task 6 |
| CI gate (drift) | Task 12 |
| Release workflow update | Task 13 |
| Retire old SKILL.md | Task 14 |
| End-to-end | Task 15 |
| Tests for generator | Task 4 (golden), Task 5 (entry-point smoke implicit), Task 15 (e2e) |
| Tests for hook | Task 11 |
| Tests for skill metadata | Task 2, Task 7 (per-cluster subset assertion) |
| Manifest schema validation | **Spec mentions snapshotting Anthropic's JSON schema; this plan does plain `jq .` validation only.** This is intentional YAGNI — the spec's "snapshot the Anthropic schema" was an idea, not a hard requirement. Adding it later is non-disruptive. |

**Placeholder check:** No "TBD"/"TODO" in tasks. The bracketed `[port from current SKILL.md §1]` notes in Task 9 are pointers to a real source file (`SKILL.md` exists today) — implementer copy-paths from there. The "implementer must read it first" notes in Task 13 reflect a real unknown (we don't have the workflow file content snapshotted in the plan), and the implementer is given the exact pattern to insert plus a pre-step to read the file.

**Type consistency:** `skillmeta.Meta`, `skillmeta.Field`, `skillmeta.Recipe`, `skillmeta.Register`, `skillmeta.All`, `skillmeta.ByCluster`, `skillmeta.Reset`, `internal/rootcmd.NewRootCmd`, `cmd/gen-skill-docs.renderSkill` — all consistent across tasks. Per-cluster `convertCheatsheet` is private and identical across each cluster (intentional copy-paste, since each cluster's `Field` type is its own per-package type).

**Scope:** Single cohesive plugin landing. No subsystem decomposition needed.
