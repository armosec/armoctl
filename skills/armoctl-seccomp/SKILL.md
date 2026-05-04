---
name: armoctl-seccomp
description: Generated seccomp profiles — list and generate profiles per workload. Use to restrict syscalls to those observed at runtime.
---

# armoctl-seccomp

Same model as network-policies but for seccomp: ARMO records the syscall set at runtime and emits a tight allow-list profile.

## Commands

- `fields` — Print the seccomp profiles resource cheatsheet
- `generate` — Generate seccomp profiles for specified workloads
  - Flags:
    - `--wlid` (stringSlice) Workload ID (repeatable, required)
- `list` — List seccomp profiles

## Resource fields

### profiles

| Field | Description |
|---|---|
| `name` | Profile name. |
| `namespace` | Kubernetes namespace. |
| `cluster` | Cluster name. |
| `kind` | Resource kind. |
| `containerName` | Container the profile applies to. |

## Field semantics

**`containerName`** — Container the profile applies to. Profiles are container-level, which is more precise than pod-level but requires one profile per container.

## Recipes

### Generate a seccomp profile for a workload

```
armoctl seccomp generate --wlid <workload-id>
```

