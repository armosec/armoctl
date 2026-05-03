---
name: armoctl-vulns
description: ARMO vulnerability triage — list CVEs, find affected images/hosts/workloads, check runtime relevance, manage exception policies. Use when the user is investigating package vulnerabilities, container CVEs, or remediation prioritization.
---

# armoctl-vulns

The `vulns` cluster covers the runtime + scan vulnerability surface. The most important triage axis is `isRelevant`: ARMO observes which packages are actually loaded in running workloads, so a Critical CVE in dormant code is a much lower priority than the same CVE in an in-use library. Always filter by isRelevant when scoping urgent work.

## Commands

- `components` — List components/packages with vulnerability counts
- `cves` — List vulnerabilities (CVEs)
  - Flags:
    - `--severity` (string) Filter by severity (Critical|High|Medium|Low)
- `exceptions` — Vulnerability exception policies
- `exceptions create` — Create a vulnerability exception policy
  - Flags:
    - `--cluster` (string) Cluster name
    - `--container` (string) Container name
    - `--cve` (stringArray) CVE to except (repeatable, required)
    - `--expires` (string) Expiration date (RFC3339)
    - `--kind` (string) Workload kind
    - `--name` (string) Policy name
    - `--namespace` (string) Namespace
    - `--reason` (string) Reason for the exception
    - `--workload` (string) Workload name
- `exceptions delete [policy-name]` — Delete a vulnerability exception policy by name
- `exceptions list` — List vulnerability exception policies
- `exceptions update` — Update an existing vulnerability exception policy
  - Flags:
    - `--cluster` (string) Cluster name
    - `--container` (string) Container name
    - `--cve` (stringArray) CVE to except (repeatable, required)
    - `--expires` (string) Expiration date (RFC3339)
    - `--guid` (string) Policy GUID (required)
    - `--kind` (string) Workload kind
    - `--name` (string) Policy name
    - `--namespace` (string) Namespace
    - `--reason` (string) Reason for the exception
    - `--workload` (string) Workload name
- `fields [scope]` — Print the vulns resource cheatsheet (optionally filtered by scope)
- `history` — Vulnerability count over time (overtime view)
- `hosts` — List hosts with vulnerability counts
- `images` — List images with vulnerability counts
- `scan` — Trigger a vulnerability scan for the given workload(s)
  - Flags:
    - `--wlid` (stringSlice) Workload ID to scan (repeatable)
- `severity` — Vulnerability counts grouped by severity
- `top` — Top vulnerabilities (used by the weekly report)
  - Flags:
    - `--severity` (string) Filter by severity
- `workloads` — List workloads with vulnerability counts

## Resource fields

### components

| Field | Description |
|---|---|
| `name` | Component (package) name. |
| `version` | Component version. |
| `packageType` | Package type (go-module, npm, ...). |
| `fixVersions` | Versions that fix known CVEs. |
| `criticalCount` | Count of critical CVEs. |
| `highCount` | Count of high CVEs. |
| `workloadsCount` | Workloads using this component. |
| `imagesCount` | Images containing this component. |
| `hasHotCVE` | Whether the component has a 'hot' CVE. |

### cves

| Field | Description |
|---|---|
| `name` | CVE name (e.g. GHSA-... or CVE-...). |
| `id` | Stable identifier. |
| `severity` | critical \| high \| medium \| low \| unknown. |
| `severityScore` | Numeric severity score. |
| `exploitable` | Known to be exploitable in the wild. |
| `isRelevant` | ARMO relevance signal: actually loaded at runtime. |
| `discoveredDate` | RFC3339 first-seen time. |
| `componentInfo` | Component context (name+version). |
| `cvssInfo` | Full CVSS info. |

### hosts

| Field | Description |
|---|---|
| `hostName` | Host name. |
| `hostType` | Host type (kubernetes/ec2/...). |
| `accountName` | Cloud account name. |
| `region` | Cloud region. |
| `kernelVersion` | Kernel version of the host. |

### images

| Field | Description |
|---|---|
| `repository` | Image repository. |
| `tag` | Image tag. |
| `registry` | Image registry. |
| `digest` | Image digest. |
| `lastScanTime` | RFC3339 time of the most recent scan. |
| `criticalCount` | Count of critical CVEs in the image. |
| `highCount` | Count of high CVEs. |
| `mediumCount` | Count of medium CVEs. |
| `lowCount` | Count of low CVEs. |
| `clusters` | Clusters where this image runs. |
| `namespaces` | Namespaces where this image runs. |

### workloads

| Field | Description |
|---|---|
| `wlid` | Workload ID; primary identifier. |
| `name` | Workload name (deployment / statefulset / etc.). |
| `namespace` | Kubernetes namespace. |
| `kind` | Resource kind. |
| `cluster` | Cluster name. |
| `lastScanTime` | RFC3339 time of the most recent scan. |
| `imagesCount` | Number of images in the workload. |
| `criticalCount` | Count of critical CVEs across the workload. |
| `highCount` | Count of high CVEs. |
| `mediumCount` | Count of medium CVEs. |
| `lowCount` | Count of low CVEs. |

## Field semantics

**`fixVersions`** — Versions that fix known CVEs. Empty means no fix available upstream — don't suggest 'upgrade' as a remediation in that case.

**`isRelevant`** — Runtime-loaded vs. dormant on disk. Critical for triage: a Critical CVE in dormant code is much lower priority than the same CVE in an in-use library. Filter with `--query '.items[] | select(.attributes.isRelevant == true)'`.

**`severity`** — ARMO severity (critical | high | medium | low | unknown), not raw CVSS — already adjusted for runtime context and exception policies.

## Recipes

### Critical CVEs that are actually in use

```
armoctl vulns cves --severity Critical --query '.items[] | select(.attributes.isRelevant == true)'
```

### Create an exception for a CVE in a workload

```
armoctl vulns exceptions create --name 'CVE-2024-12345 exception' --cve CVE-2024-12345 --workload my-app --namespace default --reason 'Planned remediation' --dry-run
```

