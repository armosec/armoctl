---
name: armoctl-foo
description: Description for the foo cluster.
---

# armoctl-foo

Summary paragraph for foo.

## Commands

- `get <guid>` — Get a foo by GUID
- `list` — List foos
  - Flags:
    - `--page-size` (int) Page size
    - `--severity` (string) Filter by severity

## Mutations

Any command that writes (create, update, delete, status changes, …) follows the safety contract of the `armoctl` skill:

1. Run it with `--dry-run` and show the user what will change.
2. Ask the user to confirm and wait for an explicit "yes". The original request is not confirmation.
3. After the user confirms, re-run the identical command with `--yes` in place of `--dry-run`. If any argument changes, dry-run and confirm again.

This holds even when the host auto-approves `armoctl` commands.

## Resource fields

### foo

| Field | Description |
|---|---|
| `guid` | Unique identifier |
| `severity` | Severity level |

## Field semantics

**`severity`** — Severity is post-policy, not raw CVSS.

## Recipes

### List Critical foos

```
armoctl foo list --severity Critical
```

