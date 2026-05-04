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

Note: The smoke script always builds a fresh binary from your working tree, so you don't need to have `armoctl` pre-installed. This ensures you're testing your branch, not a stale system install.

## What it checks

For each of the 13 clusters: one or more read-only commands plus selected dry-run mutations. Pass criteria for each check:

1. Exit code 0
2. Stdout is non-empty
3. Stdout parses as JSON

The smoke does NOT assert on specific data — empty results (`{"items":[],"total":0}`) count as passing because the goal is to verify the API/CLI path works, not that the tenant has data.

## Check outcomes

- **PASS**: Exit code 0, non-empty stdout, valid JSON
- **FAIL**: Exit code non-zero (not a transient error), empty stdout, or non-JSON stdout
- **SKIP**: The check encountered a known transient backend issue and is not counted as a failure. Examples:
  - HTML response in stdout (Cloudflare/nginx error page — usually means the feature is unconfigured for this tenant)
  - `context deadline exceeded` in stderr (request timeout)
  - Cloudflare HTML error page in stderr
  - `504 Gateway Time-out` in stderr

Transient backend errors (timeouts, Cloudflare/504 errors) are skipped rather than failed because they do not indicate a CLI bug. **400 Bad Request is treated as a real failure** since it may indicate invalid request construction in the CLI.

## Dry-run mutations included

Each mutation is exercised with `--dry-run` so no real data is written. The pass criteria is the same as read-only checks (exit 0 + JSON output):

| Cluster | Command |
|---------|---------|
| vulns | `vulns exceptions create --dry-run --name smoke-test-exception --cve CVE-0000-00000 --cluster smoke-test-not-real` |
| posture | `posture exceptions create --dry-run --name smoke-test-exception --control C-0001 --cluster smoke-test-not-real` |
| risks | `risks exceptions create --dry-run --risk-id R-0001 --reason 'smoke test' --cluster smoke-test-not-real` |
| runtime-policies | `runtime-policies create --dry-run --name smoke-test-policy` |

## What it doesn't check

- Mutations against the live API (no risk of accidental data changes)
- Commands that require resource-specific GUIDs (e.g. `incidents alerts <guid>`, `repo-posture failed-controls --report-guid <guid>`)
- The entire `integrations` cluster: all write-side surfaces (`alert-channels`, `siem` have no `list`; `jira projects` is tenant-dependent). The smoke test deliberately skips integrations; mutations would require real downstream targets.
- `repo-posture resources`, `repo-posture files`, `repo-posture failed-controls`: these require a `--report-guid` not available without a connected repo

## CI integration

To run in GitHub Actions, add a workflow that uses repository secrets `ARMO_CUSTOMER_GUID`, `ARMO_ACCESS_KEY`, `ARMO_API_BASE_URL` and calls `./scripts/smoke.sh`. Recommended trigger: manual (`workflow_dispatch`) or scheduled (`schedule:`) — not on every PR, since it consumes API quota.
