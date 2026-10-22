---
type: reality
status: active
owner: armosec
scope: repo
related_code:
  - .claude-plugin/plugin.json
  - .claude-plugin/marketplace.json
  - gemini-extension.json
  - hooks/hooks.json
  - hooks/session-start.sh
  - cmd/gen-skill-docs/render.go
  - skills/armoctl/SKILL.md
---

# Agent plugins (Claude Code, GitHub Copilot CLI, Gemini CLI)

One layout in this repo serves three agent hosts. Nothing is forked per host: the
skills, the hook script, and the hook config are shared.

| Host | Install | Reads |
|------|---------|-------|
| Claude Code | `/plugin marketplace add armosec/armoctl` → `/plugin install armoctl@armosec` | `.claude-plugin/{marketplace,plugin}.json`, `skills/`, `hooks/hooks.json` |
| GitHub Copilot CLI | `copilot plugin marketplace add armosec/armoctl` → `copilot plugin install armoctl@armosec` | Same files as Claude Code (verified on Copilot CLI 1.0.102) |
| Gemini CLI | `gemini extension install https://github.com/armosec/armoctl` | `gemini-extension.json`, `skills/`, `hooks/hooks.json` |

## Copilot CLI compatibility

Verified on Copilot CLI 1.0.102 (2026-10-22):

- Copilot looks for `plugin.json` at `.plugin/`, the root, `.github/plugin/`, then
  `.claude-plugin/`, and for `marketplace.json` in the same places. With neither of the
  earlier locations present, it uses the Claude manifests. **Adding a root `plugin.json`
  or a `.github/plugin/plugin.json` would shadow `.claude-plugin/plugin.json` for Copilot**
  — keep the versions in sync if you ever add one.
- Copilot auto-discovers `hooks/hooks.json` and accepts the Claude format (nested
  `SessionStart` → `hooks[]` → `{"type":"command","command":…}`).
- In the hook process Copilot exports `CLAUDE_PLUGIN_ROOT`, `PLUGIN_ROOT`,
  `COPILOT_PLUGIN_ROOT` (all the plugin root), `COPILOT_HOOK_DIR`, and
  `COPILOT_PLUGIN_DATA`. The hook's cwd is the plugin root. `extensionPath` is unset, so
  `${CLAUDE_PLUGIN_ROOT}${extensionPath}/hooks/session-start.sh` resolves.
- Copilot reads only the top-level `additionalContext` from SessionStart output; Claude
  Code and Gemini CLI read `hookSpecificOutput.additionalContext`. The hook emits both.
- Hook failures and timeouts (default 30s) are fail-open in Copilot.

## SessionStart hook

`hooks/session-start.sh` resolves the plugin root from its own path, not from host
variables, and reads the pinned version from `.claude-plugin/plugin.json`.

| Installed binary | Action |
|------------------|--------|
| Missing | Install the plugin's version into `~/.local/bin` via `install.sh`; emit context telling the agent it was installed (or that `~/.local/bin` is not on `PATH`, or that the install failed). |
| Release build, same version | Nothing (no output). |
| Release build, different version | `armoctl update`; emit context only if that fails. |
| Non-release build (`dev`, `v0.0.13-dirty`, `v0.0.12-4-gabc123`) | Nothing, a note on stderr. Avoids replacing a developer's local build. |

The version comes from `armoctl version` (stderr), not `--version`: released binaries up
to v0.0.13 print `unknown (built from source)` for `--version`. From the next release
`--version` reports the real version too (`fang.WithVersion` in `main.go`).

stdout is reserved for the hook JSON; installer and `armoctl update` output go to
stderr. The hook always exits 0.

## Mutation contract in skills

The root skill and every generated cluster skill (`## Mutations`, rendered by
`cmd/gen-skill-docs`) tell the agent: dry-run, show the change, wait for explicit user
confirmation, then re-run the identical command with `--yes`. The rule is repeated in the
cluster skills because agents often load only the cluster skill.

## Version consistency

`plugin_manifests_test.go` (run by PR CI via `go test ./...`) fails if the manifests are
invalid JSON or their versions differ. The release workflow (`pr-merged.yaml`) bumps all
three together.
