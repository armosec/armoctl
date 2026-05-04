---
name: armoctl-posture
description: Kubernetes posture scanning — controls, frameworks, exceptions. Use when assessing compliance posture (NSA, MITRE, etc.) or managing posture exception policies.
---

# armoctl-posture

Posture is config-time scanning of K8s resources against control frameworks. A 'failed control' means a resource violates a rule from a framework like NSA-CISA. Exception policies suppress specific (control × resource) pairs.

## Commands

- `controls` — List posture controls
  - Flags:
    - `--framework` (string) Filter by framework name
- `exceptions` — Posture exception policies
- `exceptions create` — Create a posture exception policy
  - Flags:
    - `--cluster` (string) Cluster name
    - `--container` (string) Container name
    - `--control` (stringArray) Control ID to except (repeatable, required)
    - `--expires` (string) Expiration date (RFC3339)
    - `--kind` (string) Workload kind
    - `--name` (string) Policy name
    - `--namespace` (string) Namespace
    - `--reason` (string) Reason for the exception
    - `--workload` (string) Workload name
- `exceptions delete [policy-name]` — Delete a posture exception policy by name
- `exceptions list` — List posture exception policies
- `fields [scope]` — Print the posture resource cheatsheet (optionally filtered by scope)
- `frameworks` — List compliance frameworks
- `resources` — List resources affected by posture failures

## Resource fields

### controls

| Field | Description |
|---|---|
| `name` | Control name. |
| `id` | Control ID (e.g. C-0001). |
| `severity` | Severity bucket. |
| `complianceScore` | Per-control compliance 0..1. |
| `framework` | Owning framework. |
| `status` | passed \| failed \| skipped. |
| `scoreFactor` | Weighting factor for scoring. |

### exceptions

| Field | Description |
|---|---|
| `guid` | Policy GUID. |
| `name` | Policy name. |
| `policyType` | postureExceptionPolicy. |
| `creationTime` | RFC3339 creation. |
| `actions` | ["alertOnly"]. |

### frameworks

| Field | Description |
|---|---|
| `name` | Framework name (e.g. NSA, MITRE). |
| `complianceScore` | Aggregate compliance score 0..1. |
| `totalControls` | Total controls in the framework. |
| `failedControls` | Failed control count. |
| `passedControls` | Passed control count. |
| `skippedControls` | Skipped control count. |
| `lastRun` | RFC3339 last scan time. |

### resources

| Field | Description |
|---|---|
| `name` | Resource name. |
| `namespace` | Kubernetes namespace. |
| `kind` | Resource kind. |
| `cluster` | Cluster name. |
| `complianceScore` | 0..1 compliance score. |
| `failedControlsCount` | Failing control count for this resource. |
| `warningControlsCount` | Warning control count. |
| `totalControlsCount` | Total controls evaluated. |

## Field semantics

**`framework`** — Owning framework name (NSA, MITRE, ArmoBest, etc.). A single control can belong to several frameworks.

**`id`** — Stable control identifier (e.g. C-0001). Prefer this over name when scripting — names can change between framework versions.

## Recipes

### List controls for a specific framework

```
armoctl posture controls --framework NSA
```

