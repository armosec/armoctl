# armoctl Claude Plugin — Design

**Date:** 2026-05-03
**Status:** Draft, pending user approval
**Resolves:** [shared-designs-and-docs/armoctl-agent-bridge/2026-05-01-followups.md](../../../../shared-designs-and-docs/armoctl-agent-bridge/2026-05-01-followups.md) §3

## Goal

Promote the existing `SKILL.md` into a proper Claude Code plugin so users can discover and install armoctl with one command, and so the skill content stays in sync with the binary it documents. Ship the plugin from the same repo and same release tag as the binary, with a SessionStart hook that keeps the binary present and up-to-date automatically.

## Non-goals

- MCP server wrapping armoctl (separate, larger project — agents will continue to invoke armoctl via Bash for now).
- Cursor and opencode platform support (additive after Claude Code + Gemini ships).
- A new auth/configure UX — `armoctl configure` and the env-var path are unchanged.
- Refactoring the cobra command tree.

## Architecture

The plugin is a self-contained subtree inside the existing `armoctl` repo, distributed via a self-hosted marketplace tied to the same git tags as the binary. Two manifests live in the repo (Claude Code's `plugin.json` under `.claude-plugin/`, Gemini CLI's `gemini-extension.json` at the repo root), both pointing at a single shared `skills/` directory. A SessionStart hook ensures the `armoctl` binary is present and at the same version as the plugin. Per-cluster skills are auto-generated from the cobra command tree plus curated metadata in each `cmd/<cluster>/skill.go`.

Why a single repo: the plugin and the binary it wraps must move in lockstep. A `v0.1.0` tag produces a `v0.1.0` binary AND a `v0.1.0` plugin, eliminating the drift class entirely. The existing `pr-merged.yaml` release workflow is extended to bump `plugin.json` `version` along with the binary version.

## Repository layout

```
armoctl/
├── .claude-plugin/
│   ├── plugin.json              # Claude Code plugin manifest
│   └── marketplace.json         # Self-hosted marketplace listing
├── gemini-extension.json        # Gemini CLI manifest (root, sibling of plugin.json)
├── skills/
│   ├── armoctl/SKILL.md         # Root: setup, output contract, safety, recipes index
│   ├── armoctl-incidents/SKILL.md
│   ├── armoctl-vulns/SKILL.md
│   ├── armoctl-posture/SKILL.md
│   ├── armoctl-risks/SKILL.md
│   ├── armoctl-runtime-rules/SKILL.md
│   ├── armoctl-runtime-policies/SKILL.md
│   ├── armoctl-network-policies/SKILL.md
│   ├── armoctl-seccomp/SKILL.md
│   ├── armoctl-attack-chains/SKILL.md
│   ├── armoctl-inventory/SKILL.md
│   ├── armoctl-repo-posture/SKILL.md
│   ├── armoctl-integrations/SKILL.md
│   └── armoctl-cloud-accounts/SKILL.md
├── hooks/
│   └── session-start.sh         # binary presence + version check
├── cmd/
│   ├── gen-skill-docs/main.go   # generator
│   └── <cluster>/skill.go       # per-cluster curated metadata (NEW file in each cluster)
└── install.sh                   # already exists; reused by the hook
```

The existing repo-root `SKILL.md` is replaced by `skills/armoctl/SKILL.md` (same content, trimmed to the root-skill scope). Cluster-specific recipes move into the appropriate per-cluster skill.

## Components

### 1. Claude Code manifest — `.claude-plugin/plugin.json`

```json
{
  "name": "armoctl",
  "version": "0.1.0",
  "description": "ARMO security platform CLI as a Claude skill — incidents, vulnerabilities, posture, risks, attack chains, runtime, network, integrations.",
  "author": { "name": "ARMO", "email": "support@armosec.io", "url": "https://www.armosec.io" },
  "homepage": "https://github.com/armosec/armoctl",
  "repository": "https://github.com/armosec/armoctl",
  "license": "Apache-2.0",
  "keywords": ["security", "kubernetes", "cli", "ecs", "armo", "vulnerabilities", "posture"],
  "skills": "./skills/",
  "hooks": "./hooks/"
}
```

### 2. Self-hosted marketplace — `.claude-plugin/marketplace.json`

```json
{
  "name": "armosec",
  "description": "ARMO official Claude plugin marketplace",
  "owner": { "name": "ARMO", "email": "support@armosec.io" },
  "plugins": [
    {
      "name": "armoctl",
      "description": "ARMO security platform CLI as a Claude skill",
      "version": "0.1.0",
      "source": "./",
      "author": { "name": "ARMO", "email": "support@armosec.io" }
    }
  ]
}
```

Users install with:
```
/plugin marketplace add armosec/armoctl
/plugin install armoctl@armosec
```

### 3. Gemini CLI manifest — `gemini-extension.json`

Root-level sibling. Same `skills/` directory, different loader contract (Gemini activates skills via `activate_skill` rather than auto-loading by description match). The skill markdown is platform-neutral; only the manifest differs.

### 4. Root skill — `skills/armoctl/SKILL.md`

Hand-written, ~80 lines. Sections:

1. **Setup** — `armoctl configure`, env vars (`ARMO_CUSTOMER_GUID`, `ARMO_ACCESS_KEY`, `ARMO_API_BASE_URL`), config + audit log paths.
2. **Output contract** — list/get/mutation JSON shapes; `--full`, `--fields`, `--query` flags.
3. **Safety contract** — `--dry-run`, `--yes`, TTY confirmation, audit log location.
4. **Error model** — exit codes (`CodeBadInput`, `CodeNotFound`, etc.), `RequestID` field for support escalation.
5. **Skill index** — one line per cluster pointing at the per-cluster skill name. The agent's description-matcher does the actual routing; this section is a human-readable map.

This skill is **always loaded** when the user asks anything armoctl-related. It is the contract; per-cluster skills are reference material.

### 5. Per-cluster skills — `skills/armoctl-<cluster>/SKILL.md`

**Auto-generated.** Each file is structured:

```markdown
---
name: armoctl-<cluster>
description: <curated, drives skill matching>
---

# armoctl <cluster>

<one-paragraph summary, curated>

## Commands

<auto-generated from cobra: each subcommand with one-line summary + flag table>

## Resource fields

<table from Cheatsheet() — field name, one-line label>

## Field semantics

<from FieldNotes — only fields where domain context matters; not exhaustive>

## Recipes

<curated worked examples>
```

### 6. Curation surface — `cmd/<cluster>/skill.go`

Each cluster gets a new `skill.go` next to its existing `fields.go`. It exports a `SkillMeta` struct that the generator consumes:

```go
package vulns

import "github.com/armosec/armoctl/internal/skillmeta"

var Skill = skillmeta.Meta{
    Name: "armoctl-vulns",
    Description: "Vulnerability triage — CVEs, packages, hosts, images, in-use detection, exception policies.",
    Summary: `Use when the user is investigating CVEs, container image vulns, or vulnerable packages...`,
    FieldNotes: map[string]string{
        "inUse": "Runtime-loaded vs. dormant on disk. Critical for triage: a Critical CVE in dormant code is much lower priority than the same CVE in an in-use library.",
        "fixVersion": "Empty string means no fix available upstream — don't suggest 'upgrade' as a remediation in that case.",
        "severity": "ARMO severity, not raw CVSS — already adjusted for runtime context and exception policies.",
    },
    Recipes: []skillmeta.Recipe{
        {
            Title: "Find Critical CVEs that are actually in use",
            Body:  "armoctl vulns list --severity Critical --query '.items[] | select(.attributes.inUse == true)'",
        },
    },
}
```

`internal/skillmeta` defines the shared types and a process-wide registry. Each cluster's `init()` calls `skillmeta.Register(Skill)` so the generator can iterate the registry without taking a static dependency on every cluster package by name. `cmd/gen-skill-docs/main.go` does a single side-effect import of `cmd` (which already imports every cluster) to populate the registry, then walks it.

A unit test per cluster asserts that `FieldNotes` keys are a subset of `Cheatsheet()` keys (no orphan annotations referencing fields that don't exist).

### 7. SessionStart hook — `hooks/session-start.sh`

```bash
#!/usr/bin/env bash
set -e

PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-$(dirname "$(dirname "$0")")}"
PLUGIN_VERSION="$(jq -r .version "$PLUGIN_ROOT/.claude-plugin/plugin.json")"

INSTALLED=""
if command -v armoctl >/dev/null 2>&1; then
    INSTALLED="$(armoctl --version 2>/dev/null | awk '{print $NF}')"
fi

if [ -z "$INSTALLED" ]; then
    echo "armoctl not found — installing v${PLUGIN_VERSION}…" >&2
    if ! curl -fsSL https://armoctl.armosec.io/install.sh | sh; then
        echo "armoctl install failed; the armoctl skill will not work this session." >&2
        echo "Install manually: curl -fsSL https://armoctl.armosec.io/install.sh | sh" >&2
        exit 0   # do not block session start
    fi
elif [ "v$INSTALLED" != "v$PLUGIN_VERSION" ] && [ "$INSTALLED" != "$PLUGIN_VERSION" ]; then
    echo "armoctl ${INSTALLED} differs from plugin ${PLUGIN_VERSION} — running 'armoctl update'…" >&2
    armoctl update || echo "armoctl update failed; continuing with ${INSTALLED}." >&2
fi
```

Properties:
- Idempotent: cheap (`armoctl --version`) when already in sync.
- Non-blocking on failure: prints actionable error, exits 0 so the session still starts.
- Uses the existing `install.sh` (already published by the release workflow) and `armoctl update` subcommand.

### 8. Generator — `cmd/gen-skill-docs/main.go`

A small Go program. Imports `root.Cmd()` (the configured cobra root) and walks its children. For each top-level cluster command:

1. Look up the cluster's `Skill` metadata via a registry (each cluster's `init()` registers itself with `internal/skillmeta`).
2. Walk subcommands, emit a "Commands" section: `name | one-line | flags`.
3. Emit a "Resource fields" table from the cluster's existing `Cheatsheet()`.
4. Emit a "Field semantics" subsection from `Skill.FieldNotes`.
5. Emit "Recipes" from `Skill.Recipes`.
6. Write to `skills/armoctl-<cluster>/SKILL.md` with the appropriate frontmatter.

Wired up as `make skill-docs`. CI runs `make skill-docs && git diff --exit-code skills/` so a flag added without regeneration fails the build.

### 9. Release / CI integration

The existing `.github/workflows/pr-merged.yaml` auto-bumps the patch version and tags. Two additions:

1. **Pre-tag step:** rewrite `.claude-plugin/plugin.json` and `.claude-plugin/marketplace.json` `version` fields to match the new tag. Commit alongside the version bump.
2. **Pre-tag check:** `make skill-docs && git diff --exit-code skills/` to block release on stale generated docs.

The marketplace is consumed straight from the git tag — Claude Code clones at the tag the user pinned (`armoctl@v0.1.0`) or follows the default branch for `@latest`.

## Data flow

### First-time install

1. User runs `/plugin marketplace add armosec/armoctl` — Claude Code clones the repo.
2. User runs `/plugin install armoctl@armosec` — manifest is loaded, skills are indexed by their `description` frontmatter, hook is registered.
3. Next session start — hook runs, no binary found, calls `install.sh`, binary lands at `~/.armoctl/bin/armoctl` and PATH is augmented.
4. Session loop — user asks "show me critical incidents," the description-matcher selects `armoctl-incidents`, agent runs `armoctl incidents list --severity Critical`.

### Plugin upgrade (v0.1.0 → v0.1.1)

1. `/plugin update armoctl` — manifest now at v0.1.1.
2. SessionStart — version mismatch detected, hook runs `armoctl update`, binary now at v0.1.1.

### Developer flow

1. Add a flag or subcommand under `cmd/<cluster>/`.
2. Run `make skill-docs` — regenerates the affected `skills/armoctl-<cluster>/SKILL.md`.
3. Commit. CI gate would catch you if you forgot.

## Testing

| Layer | Approach |
|---|---|
| Generator | Golden-file tests in `cmd/gen-skill-docs/testdata/` — feed a fake command tree, regenerate, diff against golden. Add a test that runs the generator against the real `root.Cmd()` and just checks it doesn't error. |
| Cluster skill metadata | Per-cluster unit test: assert `Skill.FieldNotes` keys ⊆ `Cheatsheet()` keys, and that `Skill.Description` is non-empty. |
| Hook | Shell test (Go-driven, calls `bash hooks/session-start.sh` in temp dirs with stubbed `armoctl` binaries). Cases: binary missing, version match, version mismatch, `install.sh` fails. |
| Manifest schemas | JSON-schema validation of `plugin.json` and `marketplace.json` against published Anthropic schemas (snapshotted into `.claude-plugin/schemas/`). |
| End-to-end smoke | Manual / nightly: clean container, no `armoctl`, `claude` CLI installs the plugin, opens a session, runs `armoctl incidents list --dry-run` (or equivalent), asserts success. Out of scope for this plan; tracked as follow-up. |

## Risks & mitigations

- **Drift between binary and skills.** Mitigated by `make skill-docs` CI gate plus the in-repo coupling.
- **Hook fails on a network-partitioned machine.** Hook prints actionable error and exits 0; session still starts; user gets a clear message instead of a silent break.
- **Auto-update inside a hook is invasive.** Only `armoctl update` runs after the first install (which itself was gated on the user explicitly installing the plugin). The hook never runs arbitrary install scripts unattended after that point.
- **Skill markdown drift between Claude Code and Gemini.** Avoided by sharing one `skills/` directory across both manifests. Per-platform tool-name differences (e.g., Bash vs. shell tool) are handled in skill prose only where unavoidable.
- **`Skill` metadata living in `cmd/<cluster>/skill.go` couples agent prose to Go.** Acceptable: the curation surface is small (description, summary, ~5 field notes, ~3 recipes per cluster) and the binding makes "field documented but does not exist" a compile-time error.

## Open questions (resolve during implementation)

- Exact path the install.sh lands the binary at on Linux/macOS — confirm with the existing `install.sh` rather than re-specifying here.
- Public URL the hook should curl for `install.sh`. The release workflow publishes to `s3://armo-host-agent-publish-repository/armoctl/install.sh` with CloudFront invalidation on `/armoctl/install.sh`. Confirm the user-facing CDN host (e.g., `https://armoctl.armosec.io/install.sh` vs. the raw CloudFront domain) before pinning it in the hook.
- Whether `armoctl update` should be silent on success or print a one-liner — recommendation: silent on success, one line on upgrade.
- Gemini extension manifest format — verify against current Gemini CLI docs at implementation time; the spec assumes `gemini-extension.json` based on the mongodb plugin's layout.

## Future work (deferred)

- Submit to the public `claude-plugins-official` marketplace once the self-hosted channel has shipped one stable release.
- MCP server that exposes armoctl commands as tools (separate plan).
- Cursor and opencode platform support.
- Move the `pr-merged.yaml` release workflow to also publish the plugin tarball as a release asset (currently the marketplace pulls directly from the git tag).
