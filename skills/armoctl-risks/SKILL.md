---
name: armoctl-risks
description: Security risks (cross-cutting risk view) — list/resources/severities and exception policies. Use when working with the unified ARMO risk score, not per-domain CVE/posture findings.
---

# armoctl-risks

Risks are the unified prioritisation surface that combines vulnerability + posture + runtime signal into a single severity per (resource × risk-class). Exception policies live here too.

## Commands

- `exceptions` — Manage security-risk exception policies (risk acceptance)
- `exceptions create` — Create a security-risk exception policy (accept a risk)
  - Flags:
    - `--cluster` (string) Optional resource scope: cluster name
    - `--expires` (string) Expiration date (RFC3339)
    - `--kind` (string) Optional resource scope: workload kind
    - `--name` (string) Policy name
    - `--namespace` (string) Optional resource scope: namespace
    - `--reason` (string) Reason for accepting the risk
    - `--risk-id` (string) Security risk ID to accept (required)
    - `--workload` (string) Optional resource scope: workload name
- `exceptions delete <guid>` — Delete a security-risk exception policy by GUID
- `exceptions get <guid>` — Get a security-risk exception policy by GUID
- `exceptions list` — List security-risk exception policies
  - Flags:
    - `--risk-id` (string) Filter exceptions to those covering this security risk ID
- `exceptions update` — Update an existing security-risk exception policy
  - Flags:
    - `--expires` (string) Expiration date (RFC3339)
    - `--guid` (string) Exception policy GUID (required)
    - `--name` (string) Policy name
    - `--reason` (string) Reason
    - `--risk-id` (string) Security risk ID (required)
- `fields [scope]` — Print the risks resource cheatsheet (optionally filtered by scope)
- `list` — List security risks
  - Flags:
    - `--severity` (string) Filter by severity
- `resources` — List resources affected by a security risk
  - Flags:
    - `--risk-id` (string) Security risk ID to filter resources by (required)
- `severities` — Aggregate risks by severity

## Resource fields

### exceptions

| Field | Description |
|---|---|
| `guid` | Exception policy GUID; required for get/update/delete. |
| `name` | Human-readable policy name. |
| `policyIDs` | Security risk IDs covered by the exception (exactly one supported). |
| `reason` | Reason recorded when the risk was accepted. |
| `expirationDate` | RFC3339 expiration; null = no expiry. |
| `creationTime` | RFC3339 first-created time. |
| `createdBy` | User that created the policy. |
| `resources` | Optional resource scope (PortalDesignators). |

### resources

| Field | Description |
|---|---|
| `name` | Resource name. |
| `namespace` | Kubernetes namespace. |
| `kind` | Resource kind. |
| `cluster` | Cluster name. |
| `severity` | Highest severity for the resource. |
| `riskCount` | Number of risks affecting the resource. |

### risks

| Field | Description |
|---|---|
| `name` | Risk name. |
| `id` | Risk ID. |
| `severity` | Severity bucket. |
| `category` | Risk category. |
| `controlID` | Owning control ID. |
| `smartRemediation` | Whether smart remediation is available. |
| `creationTime` | RFC3339 first-seen time. |

## Field semantics

**`policyIDs`** — On exceptions: the risk IDs the exception applies to. Single-element in current API even though it's an array.

**`severity`** — Composite severity — already accounts for runtime context, exceptions, and exposure. Use to filter with --severity Critical|high|medium|low.

## Recipes

### List Critical risks

```
armoctl risks list --severity Critical
```

### Create an exception for a risk with an expiry date

```
armoctl risks exceptions create --risk-id <id> --reason 'planned remediation' --expires 2026-06-01T00:00:00Z --dry-run
```

