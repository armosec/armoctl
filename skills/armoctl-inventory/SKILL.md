---
name: armoctl-inventory
description: Cluster inventory — list workloads, get unique values for a field. Use to enumerate or pivot on resources before applying another command.
---

# armoctl-inventory

Inventory is the index of everything ARMO has seen. Use 'inventory list' to enumerate workloads/resources and 'inventory unique-values <field>' to discover the legal values for a given field (clusters, namespaces, kinds, etc.).

## Commands

- `fields` — Print the inventory resource cheatsheet
- `list` — List workload inventory
- `unique-values [field]` — Get unique values for an inventory field

## Resource fields

### inventory

| Field | Description |
|---|---|
| `wlid` | Workload ID. |
| `name` | Workload name. |
| `namespace` | Kubernetes namespace. |
| `kind` | Resource kind. |
| `cluster` | Cluster name. |
| `lastInventoryScanTime` | RFC3339 last scan time. |

## Field semantics

**`kind`** — K8s kind — Deployment, StatefulSet, DaemonSet, Job, CronJob, Pod. Use 'inventory unique-values kind' to confirm the spelling expected by other commands.

## Recipes

### List unique namespaces in a cluster

```
armoctl inventory unique-values namespace
```

