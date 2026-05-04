---
name: armoctl-repo-posture
description: Repository posture — IaC scanning of a connected git repo for config issues, with per-file and per-control views. Use when reviewing posture findings tied to a repo, not a live cluster.
---

# armoctl-repo-posture

Same control surface as cluster posture, but the resources are files in a connected git repo. Findings carry both file path and control ID, so they can be deep-linked back to the IaC source.

## Commands

- `failed-controls` — List failed controls per repo
  - Flags:
    - `--kind` (string) Entity kind: repo or file
    - `--report-guid` (string) Report GUID (required)
- `fields [scope]` — Print the repo-posture resource cheatsheet (optionally filtered by scope)
- `files` — List files
  - Flags:
    - `--repository` (string) Filter by repository name
- `repositories` — List repositories
- `resources` — List resources (IaC objects)

## Resource fields

### failed-controls

| Field | Description |
|---|---|
| `id` | Control ID. |
| `name` | Control name. |
| `severity` | Severity bucket. |
| `framework` | Owning framework. |
| `scoreFactor` | Weighting factor. |
| `complianceScore` | Per-control compliance 0..1. |

### files

| Field | Description |
|---|---|
| `path` | Path within the repo. |
| `name` | File name. |
| `type` | File type (yaml, terraform, ...). |
| `lastScanTime` | RFC3339 last scan. |
| `complianceScore` | 0..1 compliance score. |
| `failedControlsCount` | Failed control count for the file. |

### repositories

| Field | Description |
|---|---|
| `name` | Repository name. |
| `owner` | Owner / organization. |
| `provider` | git provider (github, gitlab, ...). |
| `branch` | Default branch. |
| `lastScanTime` | RFC3339 last scan. |
| `complianceScore` | 0..1 compliance score. |
| `failedControlsCount` | Failed control count for the repo. |

### resources

| Field | Description |
|---|---|
| `name` | Resource name. |
| `kind` | Kind (Deployment, Pod, ...). |
| `filePath` | Source file path. |
| `complianceScore` | 0..1 compliance score. |
| `failedControlsCount` | Failed control count for the resource. |

## Field semantics

**`filePath`** — Repo-relative source file path. Pair with the repo's commit SHA to deep-link to the exact line in the IaC file that caused the finding.

## Recipes

### List failed controls in a repo scan report

```
armoctl repo-posture failed-controls --report-guid <guid>
```

