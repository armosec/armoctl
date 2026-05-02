# Vulns Cluster Implementation Plan (2026-05-01)

> **For agentic workers:** Use superpowers:subagent-driven-development to execute task-by-task.

**Goal:** Add `armoctl vulns` cluster covering vulnerability list endpoints (workloads/images/components/vulnerabilities/hosts), aggregate views (top/severity/history), exceptions CRUD, and a scan mutation.

**Branch:** `feature/agent-bridge-vulns` (stacked on `feature/agent-bridge-foundation-incidents`).

**Real ARMO API shapes (verified live):**
- `POST /vulnerability_v2/{scope}/list` — scope ∈ {workload, image, component, vulnerability, host}. Body has pagination (`pageNum`/`pageSize`) plus `innerFilters`. Response: `{response:[...], total:{value,relation}, cursor}`.
- `POST /vulnerability/severity` — aggregate severity counts.
- `POST /vulnerability/topVulnerabilities` — top N.
- `POST /vulnerability/overtime` — vuln counts over time.
- `POST /vulnerability/scan` — trigger workload scan (mutation).
- `POST /vulnerability/scanResultsDetails` and `POST /vulnerability/scanResultsSumSummary` — scan-by-scan info (lower priority; defer).
- `GET /vulnerabilityExceptionPolicy` (list), `POST` (create), `PUT` (update), `DELETE` (delete).

## Tasks

Each task follows TDD (write test → fail → impl → pass → commit). Reuse the `incidents` cluster as the template. All list commands use `apiclient.ListPaged(ctx, path, nil, ListOpts{Method:"POST", Body: bodyMap, ...})`.

### Task 1 — types + cluster root + fields cheatsheet

Create `cmd/vulns/types.go`, `cmd/vulns/vulns.go`, `cmd/vulns/fields.go`, `cmd/vulns/fields_test.go`.

- `Cmd(clientFor incidents.ClientFor) *cobra.Command` — wait, ClientFor is in `cmd/incidents`. **Move ClientFor to `cmd/cliflags`** (or new package `cmd/clibase`) so other clusters can use it without depending on incidents. Adjust this in Task 1; update incidents to import the new location.
- `SummaryFields` per scope (workloads/images/components/vulnerabilities/hosts) since they have very different shapes. Provide named vars: `WorkloadSummary`, `ImageSummary`, `ComponentSummary`, `VulnerabilitySummary`, `HostSummary`.
- One `Cheatsheet()` function that returns a map[scope][]Field, plus a `FieldsCmd` that prints the requested scope (or all if no arg).

Verified field names from live probes:
- workloads: `wlid, name, workload, namespace, kind, cluster, lastScanTime, imagesCount, criticalCount, highCount, mediumCount, lowCount`
- images: `tag, digest, registry, repository, lastScanTime, clusters, namespaces, criticalCount, ...`
- components: `name, version, packageType, fixVersions, criticalCount, ..., clustersCount, namespacesCount, workloadsCount, imagesCount, hasHotCVE`
- vulnerabilities (CVEs): `name, id, severity, severityScore, exploitable, isRelevant, discoveredDate, cvssInfo, componentInfo`
- hosts: `hostName, hostType, region, kernelVersion, accountName`

### Task 2 — `vulns workloads list`

`cmd/vulns/workloads.go` + test. Path `/vulnerability_v2/workload/list`. Same pattern as incidents list but using `WorkloadSummary` defaults. No `--severity` filter for now.

### Task 3 — `vulns images list`

`cmd/vulns/images.go` + test. Path `/vulnerability_v2/image/list`.

### Task 4 — `vulns components list`

`cmd/vulns/components.go` + test. Path `/vulnerability_v2/component/list`.

### Task 5 — `vulns cves list`

`cmd/vulns/cves.go` + test. Path `/vulnerability_v2/vulnerability/list`. Add `--severity` flag → maps to body `innerFilters: [{severity: "critical|high|..."}]` (one filter group).

### Task 6 — `vulns hosts list`

`cmd/vulns/hosts.go` + test. Path `/vulnerability_v2/host/list`.

### Task 7 — `vulns top`

`cmd/vulns/top.go` + test. POST `/vulnerability/topVulnerabilities` with body `{}`. Single object response → use `cli.PostJSON`. Add `--limit` flag mapped to body if API supports it (probe; if not, just call).

### Task 8 — `vulns severity`

`cmd/vulns/severity.go` + test. POST `/vulnerability/severity` with body `{}`. Single object response.

### Task 9 — `vulns history`

`cmd/vulns/history.go` + test. POST `/vulnerability/overtime`. Use ListPaged with POST since body might need pagination.

### Task 10 — `vulns scan` (mutation)

`cmd/vulns/scan.go` + test. POST `/vulnerability/scan` with body `{wlids: [...]}`. Uses `safety.Wrap`. Args: `armoctl vulns scan --wlid <wlid> [--wlid <wlid>]`. Builds `body["wlids"] = wlids`.

### Task 11 — `vulns exceptions list`

`cmd/vulns/exceptions_list.go` + test. GET `/vulnerabilityExceptionPolicy`. Use `cli.GetJSON` and `output.Get` (or `output.List` if response is paginated — probe to see).

### Task 12 — `vulns exceptions create`

`cmd/vulns/exceptions_create.go` + test. POST `/vulnerabilityExceptionPolicy` with body shape matching `armotypes.VulnerabilityExceptionPolicy`. Mutation. Args: `--name`, `--cve` (repeatable), `--reason`, `--expiry`. Build the body. Use `safety.Wrap`.

### Task 13 — `vulns exceptions update`

`cmd/vulns/exceptions_update.go` + test. PUT `/vulnerabilityExceptionPolicy` with body containing the policy GUID + updated fields. Mutation.

### Task 14 — `vulns exceptions delete`

`cmd/vulns/exceptions_delete.go` + test. DELETE `/vulnerabilityExceptionPolicy?policyGUID=<guid>`. Mutation. Args: `<policyGUID>` positional.

### Task 15 — Wire `vulns` cluster into root + e2e test

Modify `root.go` to add `rootCmd.AddCommand(vulnscmd.Cmd(clientFor))`. Add `cmd/vulns/vulns_e2e_test.go` exercising 2-3 list flows + the scan dry-run via httptest mux.

### Task 16 — Live smoke against api-dev.armosec.io

Run read-only smoke battery: each list scope, top, severity, history, exceptions list. Document any upstream issues (like the 504s seen on incidents resolve).

### Task 17 — Push branch + open PR (base=feature/agent-bridge-foundation-incidents)

```bash
git push -u origin feature/agent-bridge-vulns
gh pr create --base feature/agent-bridge-foundation-incidents \
  --title "feat: vulns cluster (workloads/images/components/cves/hosts/top/severity/history/scan/exceptions)" \
  --body "$(cat <<'BODY'
## Summary
Adds the vulns cluster: 5 scoped list commands, 3 aggregate views, scan mutation, exceptions CRUD.
Stacked on PR #12 (foundation+incidents).
BODY
)"
```

## Self-review checklist

After implementation:
- [ ] All tests green.
- [ ] `go build ./...` clean.
- [ ] `armoctl vulns --help` shows all subcommands.
- [ ] Live read-only smoke works for at least 4 of 5 scopes plus severity.
- [ ] PR description lists working live endpoints + any upstream issues.
