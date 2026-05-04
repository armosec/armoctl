---
name: armoctl
description: ARMO security platform CLI — JSON-first agent-friendly access to runtime incidents, vulnerabilities, posture, risks, attack chains, inventory, network policies, seccomp, runtime rules/policies, integrations, cloud accounts, and repository posture. Mutation safety with --dry-run/--yes.
---

# armoctl — ARMO Security Platform CLI

You are a security analyst with `armoctl`. It exposes 13 resource clusters as `armoctl <cluster> <subcommand>`, returns JSON by default, and wraps every mutation with a dry-run/--yes safety contract plus an audit log.

## 1. Setup

**Where to find the credentials:**
- **Customer GUID:** ARMO Platform UI → top-right account dropdown
- **Access Key:** https://cloud.armosec.io/settings/workspace/agent-access-keys (EU) or https://cloud.us.armosec.io/settings/workspace/agent-access-keys (US)

The user has three ways to authenticate. Pick whichever suits the situation:

### A. Through this chat (recommended when running inside an AI agent)

Ask the user to paste their Customer GUID and Access Key in the conversation, then run configure non-interactively. **Always read the access key from stdin** so it never appears in shell history or `ps` output:

```bash
echo "<ACCESS_KEY_PASTED_BY_USER>" | armoctl configure \
    --customer-guid "<CUSTOMER_GUID_PASTED_BY_USER>" \
    --access-key-stdin
```

This validates the credentials against the ARMO API and exits non-zero if they are rejected. After it succeeds, the user is fully configured and you don't need to repeat this in future sessions.

### B. The user runs `armoctl configure` themselves (no AI involved)

If the user prefers, they can open a terminal and run:

```bash
armoctl configure
```

That opens an interactive prompt (TUI) that walks them through Customer GUID, Access Key, and API URL. Useful when they don't want to paste secrets into chat. Once they finish, the same `~/.armoctl/config.yaml` is written and armoctl works normally afterwards.

### C. Environment variables (CI / containers / one-off shells)

```bash
export ARMO_CUSTOMER_GUID="..."
export ARMO_ACCESS_KEY="..."
export ARMO_API_BASE_URL="api.armosec.io"   # default; override for staging
```

Env vars take precedence over `~/.armoctl/config.yaml` for that shell only.

---

Credentials are stored at `~/.armoctl/config.yaml` (mode 0600). The audit log lives at `~/.armoctl/audit.log` (override via `$ARMOCTL_AUDIT_LOG`).

When armoctl reports `authentication required`, fall back to A or B before retrying.

## 2. Output contract

Every command emits JSON on stdout. Three shapes:

**List** (`armoctl <cluster> list ...`):
```json
{ "items": [...], "total": 1234, "page": 1, "pageSize": 50, "nextCursor": "..." }
```

**Get** (singular):
```json
{ "guid": "...", "name": "...", ...full-object... }
```

**Mutation** (any command that writes):
```json
{ "result": {...}, "changed": true, "dryRun": false }
```

### Token-efficient access

Each `list` returns a curated **summary projection** by default — typically 8–12 fields per resource. To override:

| Flag | Effect |
|---|---|
| `--full` | Return the raw API response (every field). |
| `--fields a,b,c.d` | Keep only these dotted paths. Applied before `--query`. |
| `--query '<gojq>'` | Run a gojq expression on the (already-projected) result before render. Operates on whichever fields survive `--fields` (or all fields if `--fields` is omitted). |

### Other format flags

`--output json|yaml|ndjson|table|csv` (default json). `--limit N` caps auto-paged lists (default 500). `--page N --page-size M` for explicit paging.

### Discovering fields

```bash
armoctl <cluster> fields           # cheatsheet for the resource
armoctl schema <resource>          # full JSON schema (where embedded)
armoctl <cluster> --help           # all subcommands
```

## 3. Safety contract

Every mutating command supports:

- `--dry-run` — builds the request, prints the would-be payload, **does not send**. Always run this first.
- `--yes` — skips confirmation. **Required** in non-interactive (agent/CI) contexts; without it, mutations exit with code 6 ("NEEDS_CONFIRM").
- Audit log entry per executed mutation (RFC3339 timestamp, command, URL, request ID).

**Rule:** for any mutation, run `--dry-run` first to inspect the payload, then re-run with `--yes` once you're confident.

## 4. Error model

Exit codes: `0 OK · 2 BadInput · 3 Auth · 4 NotFound · 5 Server · 6 NeedsConfirm · 7 Conflict`

Errors emit JSON to **stderr** (`{error, code, hint, requestId}`) so stdout stays clean for piping.

Common scenarios:
- **Exit 6** — mutation attempted without `--yes` in non-interactive context. Add `--yes` and retry.
- **Exit 3** — credentials missing or expired. Run `armoctl configure` or set `ARMO_CUSTOMER_GUID` / `ARMO_ACCESS_KEY`.
- **Exit 4** — resource GUID not found; verify with `list` first.
- **Exit 5** — upstream API error; the `requestId` in stderr can be shared with support.
- **Some endpoints require a filter.** For example, `risks resources` requires `--risk-id`; `repo-posture failed-controls` requires `--report-guid`. The error message will say what's missing.

### When confused

```bash
armoctl --help                       # top-level commands
armoctl <cluster> --help             # subcommands of a cluster
armoctl <cluster> fields             # field cheatsheet
armoctl <cluster> <cmd> --help       # flags for one command
```

## 5. Cluster index

For cluster-specific commands and field semantics, consult the matching skill:

| Cluster | Skill |
|---|---|
| Runtime incidents | `armoctl-incidents` |
| Vulnerabilities | `armoctl-vulns` |
| Posture | `armoctl-posture` |
| Risks (cross-cutting) | `armoctl-risks` |
| Attack chains | `armoctl-attack-chains` |
| Inventory | `armoctl-inventory` |
| Network policies | `armoctl-network-policies` |
| Seccomp profiles | `armoctl-seccomp` |
| Runtime rules | `armoctl-runtime-rules` |
| Runtime policies | `armoctl-runtime-policies` |
| Integrations | `armoctl-integrations` |
| Cloud accounts | `armoctl-cloud-accounts` |
| Repository posture | `armoctl-repo-posture` |

These skills are auto-loaded by description match when the user's task touches the cluster.
