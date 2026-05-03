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

