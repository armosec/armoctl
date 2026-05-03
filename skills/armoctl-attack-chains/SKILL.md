---
name: armoctl-attack-chains
description: Attack chains — multi-step kill-chain views built by ARMO from runtime + posture signal. Use when the user wants to understand how vulnerabilities chain into reachable exploit paths.
---

# armoctl-attack-chains

An attack chain links a posture weakness, a vulnerable component, and runtime context into a sequence an attacker could traverse. List view shows the highest-severity chains; details show the per-step evidence.

## Commands

- `fields` — Print the attack chains resource cheatsheet
- `list` — List attack chains

## Resource fields

### attack-chains

| Field | Description |
|---|---|
| `name` | Attack chain name. |
| `guid` | Stable identifier. |
| `creationTime` | RFC3339 first-seen time. |
| `severity` | Severity bucket. |
| `clusterName` | Cluster name. |
| `namespace` | Kubernetes namespace. |

## Field semantics

**`severity`** — Chain severity — reflects the worst-case step in the chain. Use to prioritise which chains to investigate first.

## Recipes

### List active attack chains

```
armoctl attack-chains list
```

