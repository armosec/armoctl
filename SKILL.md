---
name: armoctl
description: ARMO security platform CLI — agent-friendly access to runtime incidents, vulnerabilities, posture, risks, attack chains, inventory, network policies, seccomp, runtime rules/policies, integrations, and repository posture. JSON-first, query/projection support, mutation safety with --dry-run.
---

# armoctl — ARMO Security Platform CLI

You are a security analyst with `armoctl`, a CLI to the ARMO security platform. It exposes 14 resource clusters as `armoctl <cluster> <subcommand>`, returns JSON by default, and wraps every mutation with a dry-run/--yes safety contract plus an audit log.

This is your contract with the tool. Read it once and refer back as needed.

## 1. Setup

```bash
# One-time
armoctl configure   # prompts for customer GUID + access key + (optional) api base URL

# Or via env vars (preferred for agents)
export ARMO_CUSTOMER_GUID="..."
export ARMO_ACCESS_KEY="..."
export ARMO_API_BASE_URL="api.armosec.io"   # default; override for staging
```

Credentials are stored at `~/.armoctl/config.yaml`. The audit log lives at `~/.armoctl/audit.log` (override via `$ARMOCTL_AUDIT_LOG`).

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
| `--fields a,b,c.d` | Keep only these dotted paths. |
| `--query '<gojq>'` | Run a gojq expression on the result before render. Wins over fields. |

Example — list incidents, keep only id+severity:
```
armoctl incidents list --query '.items[] | {guid, severity, status: .attributes.incidentStatus}'
```

### Other format flags

`--output json|yaml|ndjson|table|csv` (default json). `--limit N` caps auto-paged lists (default 500). `--page N --page-size M` for explicit paging.

### Discovering fields

```
armoctl <cluster> fields           # cheatsheet for the resource
armoctl schema <resource>          # full JSON schema (where embedded)
armoctl <cluster> --help           # all subcommands
```

## 3. Mutation safety

Every mutating command supports:
- `--dry-run` → builds the request, prints the would-be payload, **does not send**. Always run this first.
- `--yes` → skips confirmation. **Required** in non-interactive (agent/CI) contexts; without it, mutations exit with code 6 ("NEEDS_CONFIRM").
- Audit log entry per executed mutation (RFC3339 timestamp, command, URL, request ID).

**Rule:** for any mutation, run `--dry-run` first to inspect the payload, then re-run with `--yes` once you're confident.

## 4. Exit codes

`0 OK · 2 BadInput · 3 Auth · 4 NotFound · 5 Server · 6 NeedsConfirm · 7 Conflict`

Errors emit JSON to **stderr** (`{error, code, hint, requestId}`) so stdout stays clean for piping.

## 5. The 14 clusters at a glance

| Cluster | Purpose | Top subcommands |
|---|---|---|
| `incidents` | Runtime threat detections | list, alerts, explain, resolve, severities |
| `vulns` | Vulnerability scanning | workloads/images/components/cves/hosts list, top, severity, history, scan, exceptions CRUD |
| `posture` | Compliance | frameworks, controls, resources, exceptions list/create/delete |
| `risks` | Prioritized security risks | list, resources, severities |
| `attack-chains` | Linked threat sequences | list |
| `inventory` | Workload inventory | list, unique-values |
| `network-policies` | Network policy artifacts | list, generate |
| `seccomp` | Seccomp profiles | list, generate |
| `cloud-accounts` | ECS connections | ecs list, connect, disconnect |
| `runtime-rules` | Detection rules | list, get, create, update, delete, evaluate |
| `runtime-policies` | Detection policies | list, create, update |
| `integrations` | Jira / SIEM / alert channels | jira projects/issue-types/fields/create-ticket, alert-channels, siem, unlink |
| `repo-posture` | IaC repo scans | repositories, files, resources, failed-controls |
| `schema` | JSON schemas | `<resource>`, `--list` |

## 6. Recipes

### 6.1 Triage runtime incidents

```bash
# 1. List today's high+critical incidents
armoctl incidents list --query '.items[] | select(.severity=="Critical" or .severity=="High")'

# 2. Drill into one incident
armoctl incidents alerts <guid> --limit 5
armoctl incidents explain <guid> --query .explanation

# 3. Resolve as false-positive (always dry-run first)
armoctl incidents resolve <guid> --dry-run
armoctl incidents resolve <guid> --yes --false-positive
```

### 6.2 Investigate a CVE across the fleet

```bash
# Top critical CVEs
armoctl vulns cves --severity Critical --limit 20 --query '.items[] | {name, exploitable, isRelevant, count: .workloadsCount}'

# Workloads with vulns, sorted by criticals
armoctl vulns workloads --limit 20 --query '.items[] | select(.criticalCount > 0) | {workload: .name, ns: .namespace, critical: .criticalCount}'

# Add an exception (always dry-run first)
armoctl vulns exceptions create --name "ignore-cve-foo" --cve CVE-2024-1234 --cluster prod --dry-run
armoctl vulns exceptions create --name "ignore-cve-foo" --cve CVE-2024-1234 --cluster prod --reason "false positive" --yes
```

### 6.3 Posture review of a cluster

```bash
armoctl posture frameworks --query '.items[] | {name, score: .complianceScore, failed: .failedControls}'
armoctl posture controls --framework NSA --limit 10 --query '.items[] | select(.status=="failed")'
armoctl posture exceptions list
```

### 6.4 Inventory & filtering

```bash
armoctl inventory list --limit 5
armoctl inventory unique-values cluster      # list every cluster name in the tenant
armoctl inventory list --limit 100 --query '.items[] | select(.cluster=="prod")'
```

### 6.5 Generate a network policy

```bash
armoctl network-policies generate --wlid wlid://cluster-prod/namespace-default/deployment-myapp --dry-run
armoctl network-policies generate --wlid wlid://cluster-prod/namespace-default/deployment-myapp --yes
```

### 6.6 Open a Jira ticket from a finding

```bash
armoctl integrations jira projects --query '.items[] | {key, name}'
armoctl integrations jira issue-types --project SEC --query '.items[].name'
armoctl integrations jira create-ticket \
  --project SEC --issue-type Bug \
  --summary "Critical CVE-2024-1234 in prod" \
  --description "Found in 3 workloads via armoctl" \
  --dry-run
# Re-run with --yes when satisfied
```

### 6.7 Risk prioritization

```bash
armoctl risks list --limit 10 --severity Critical --query '.items[] | {id, name, category}'
armoctl risks severities                              # summary counts
armoctl risks resources --risk-id <risk-guid>          # resources affected by one risk
```

## 7. Pitfalls

1. **Always pass `--limit`** on lists. Auto-paging defaults to 500; be explicit.
2. **Mutations require `--yes`** in non-interactive use. Forgetting it produces exit code 6 with a clear hint.
3. **Customer GUID scope** — every API call uses the configured customer GUID. To work across tenants, switch creds.
4. **Default summary projection drops fields.** When the field you want isn't in the JSON, add `--full` or use `--query`.
5. **Some endpoints expect a filter.** `risks resources` requires `--risk-id`; `repo-posture failed-controls` requires `--report-guid`. The error message will tell you what's missing.
6. **`incidents resolve --yes` may hit upstream gateway issues** on the dev backend — known platform-side flakiness, not a CLI bug. The `--dry-run` payload is correct.
7. **`runtime-rules evaluate` and exceptions create** require well-formed rule/CVE inputs. Use `--dry-run` to see what would be sent before running for real.

## 8. When confused

```bash
armoctl --help                       # top-level commands
armoctl <cluster> --help             # subcommands of a cluster
armoctl <cluster> fields             # field cheatsheet
armoctl <cluster> <cmd> --help       # flags for one command
```

End of skill.
