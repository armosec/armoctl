# Smoke Tests

`smoke.sh` exercises the most important read-only commands of each cluster against a live ARMO tenant. It's a high-level functional test — not a unit test — so it requires real credentials.

## Local run

```bash
export ARMO_CUSTOMER_GUID=...
export ARMO_ACCESS_KEY=...
export ARMO_API_BASE_URL=api.armosec.io   # or api.us.armosec.io for US tenants

# Run all clusters
./scripts/smoke.sh

# Run only one cluster
./scripts/smoke.sh -c vulns

# Verbose (prints output snippet per check)
./scripts/smoke.sh -v
```

## What it checks

For each of the 13 clusters: one or more read-only commands plus selected dry-run mutations. Pass criteria for each check:

1. Exit code 0
2. Stdout is non-empty
3. Stdout parses as JSON

The smoke does NOT assert on specific data — empty results (`{"items":[],"total":0}`) count as passing because the goal is to verify the API/CLI path works, not that the tenant has data.

## Dry-run mutations included

Each mutation is exercised with `--dry-run` so no real data is written. The pass criteria is the same as read-only checks (exit 0 + JSON output):

| Cluster | Command |
|---------|---------|
| vulns | `vulns exceptions create --dry-run --name smoke-test-exception --cve CVE-0000-00000` |
| posture | `posture exceptions create --dry-run --name smoke-test-exception --control C-0001` |
| risks | `risks exceptions create --dry-run --name smoke-test-exception` |
| runtime-rules | `runtime-rules create --dry-run --name smoke-test-rule` |
| runtime-policies | `runtime-policies create --dry-run --name smoke-test-policy` |

## What it doesn't check

- Mutations against the live API (no risk of accidental data changes)
- Commands that require resource-specific GUIDs (e.g. `incidents alerts <guid>`, `repo-posture failed-controls --report-guid <guid>`)
- `integrations alert-channels` and `integrations siem`: these clusters have no list subcommand (only `create`), so they are intentionally excluded from the read-only sweep
- Jira integration: `integrations jira projects` runs but a missing Jira connection counts as a failure; if your tenant doesn't use Jira, expect 1 failure (or filter it out with `-c` to skip integrations entirely)
- `repo-posture resources`, `repo-posture files`, `repo-posture failed-controls`: these require a `--report-guid` not available without a connected repo

## CI integration

To run in GitHub Actions, add a workflow that uses repository secrets `ARMO_CUSTOMER_GUID`, `ARMO_ACCESS_KEY`, `ARMO_API_BASE_URL` and calls `./scripts/smoke.sh`. Recommended trigger: manual (`workflow_dispatch`) or scheduled (`schedule:`) — not on every PR, since it consumes API quota.
