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

