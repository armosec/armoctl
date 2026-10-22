---
name: armoctl-network-policies
description: Generated NetworkPolicies — list discovered policies and generate one for a workload from observed traffic. Use to harden cluster network egress/ingress.
---

# armoctl-network-policies

ARMO observes runtime traffic and emits a least-privilege NetworkPolicy YAML for any selected workload. List shows historical policies; generate produces one on-demand.

## Commands

- `fields` — Print the network policies resource cheatsheet
- `generate` — Generate network policies for specified workloads
  - Flags:
    - `--wlid` (stringSlice) Workload ID (repeatable, required)
- `list` — List network policies

## Mutations

Any command that writes (create, update, delete, status changes, …) follows the safety contract of the `armoctl` skill:

1. Run it with `--dry-run` and show the user what will change.
2. Ask the user to confirm and wait for an explicit "yes". The original request is not confirmation.
3. After the user confirms, re-run the identical command with `--yes` in place of `--dry-run`. If any argument changes, dry-run and confirm again.

This holds even when the host auto-approves `armoctl` commands.

## Resource fields

### policies

| Field | Description |
|---|---|
| `name` | Policy name. |
| `namespace` | Kubernetes namespace. |
| `cluster` | Cluster name. |
| `kind` | NetworkPolicy kind. |
| `creationTimestamp` | RFC3339 creation. |

## Field semantics

**`kind`** — NetworkPolicy kind. Use 'inventory unique-values kind' to verify the workload kind name before generating a policy for it.

## Recipes

### Generate a network policy for a workload

```
armoctl network-policies generate --wlid <workload-id>
```

