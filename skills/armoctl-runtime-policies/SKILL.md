---
name: armoctl-runtime-policies
description: Runtime policies — bundles of rules attached to clusters/namespaces/workloads. Use to manage which detection rules apply where.
---

# armoctl-runtime-policies

A policy is a bag of runtime-rules with a binding scope (cluster, namespace, workload). When a workload runs, the union of policies that bind to it determines which rules evaluate.

## Commands

- `create` — Create a runtime policy
  - Flags:
    - `--description` (string) Policy description
    - `--enabled` (bool) Enable the policy
    - `--name` (string) Policy name (required)
    - `--policy-file` (string) Path to JSON file containing the full policy body
- `fields` — Print the runtime-policies resource cheatsheet
- `list` — List runtime policies
  - Flags:
    - `--rulesettype` (string) Filter by ruleset type (Managed or Custom)
- `update [guid]` — Update a runtime policy
  - Flags:
    - `--description` (string) Policy description
    - `--enabled` (string) Enable/disable the policy (true|false)
    - `--name` (string) Policy name
    - `--policy-file` (string) Path to JSON file containing policy updates

## Resource fields

### runtime-policies

| Field | Description |
|---|---|
| `guid` | Stable policy ID. |
| `name` | Policy name. |
| `description` | Human-readable description. |
| `enabled` | Whether the policy is active. |
| `scope` | Scope object (clusters/namespaces/workloads). |
| `creationTime` | RFC3339 creation. |

## Field semantics

**`enabled`** — Whether the policy is currently active. Disabled policies are stored but do not generate incidents.

**`scope`** — Scope object describing which clusters/namespaces/workloads this policy binds to. Most-specific binding wins on conflict.

## Recipes

### List managed runtime policies

```
armoctl runtime-policies list --rulesettype Managed
```

### List custom runtime policies

```
armoctl runtime-policies list --rulesettype Custom
```

