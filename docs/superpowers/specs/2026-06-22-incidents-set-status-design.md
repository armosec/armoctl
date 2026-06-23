# armoctl incidents set-status — Design

**Date:** 2026-06-22
**Status:** Approved (pending spec review)
**Author:** ben@armosec.io

## Problem

`armoctl incidents` can only move incidents to **Resolved**. The `resolve` command
hardcodes `"status": "Resolved"` in the request body and accepts a single GUID. A
customer (Bonial) wants to **bulk-dismiss ~13K incidents** from the CLI instead of
the UI, which is impossible today.

The backend endpoint already supports everything needed:

- `POST /runtime/incidents/changeStatus` accepts a free-form `status`, an
  `incidentsGuids []string` array, `innerFilters []map[string]string`, a
  `markedAsFalsePositive` bool, and a `searchText` **URL query parameter**.
- Valid statuses (`kdr.IncidentStatus`): `Open`, `Investigating`, `Dismissed`,
  `Resolved`. `Validate()` only requires a non-empty status plus at least one of
  `incidentsGuids` or `innerFilters`.
- Status changes are processed **asynchronously** (published to Pulsar); the handler
  also resolves `searchText` to GUIDs and enriches cluster names server-side. Bulk
  fan-out is therefore handled downstream — the CLI just sends the right request.

So no backend work is required. This is purely a CLI surface change.

## Goals

- Support setting any of the four statuses (primary driver: **Dismissed**).
- Support bulk selection at ~13K scale via both an explicit GUID set and
  filter-based selection.
- Preserve the existing `resolve` behavior and its tests exactly.

## Non-goals (YAGNI)

- Pre-flight count of affected incidents before confirming.
- Status values beyond the four defined by the backend.
- OR-across-filter-maps syntax (multiple `innerFilters` maps).
- Retroactive/historical incident handling beyond what the endpoint already does.

## Command surface

```text
armoctl incidents set-status [guid...] --status <Open|Investigating|Dismissed|Resolved> [flags]
armoctl incidents resolve <guid> [--false-positive]      # unchanged; now an alias
```

`set-status` flags:

| Flag | Meaning |
|------|---------|
| `--status string` | **Required.** One of `Open\|Investigating\|Dismissed\|Resolved`. Case-insensitive input, normalized to canonical form. |
| `--filter key=value` | Repeatable. Builds a single `innerFilters` map (AND across filters). |
| `--search string` | Free-text; sent as the `searchText` URL query parameter. |
| `--stdin` | Read additional GUIDs from stdin (whitespace/newline separated). |
| `--false-positive` | Sets `markedAsFalsePositive` (carried over from `resolve`). |
| `[guid...]` | Positional GUIDs. |

## Architecture

One shared internal helper owns the request build, status validation, and
`safety.Wrap`. Both commands route through it, giving a single code path for the
`changeStatus` call and audit logging.

```text
cmd/incidents/
  changestatus.go   # NEW: runStatusChange() helper + status normalization/validation
  setstatus.go      # NEW: SetStatusCmd — flags → statusChangeOpts → runStatusChange
  resolve.go        # REFACTORED: thin wrapper → runStatusChange(status=Resolved)
  incidents.go      # +c.AddCommand(SetStatusCmd(clientFor))
```

Helper shape:

```go
type statusChangeOpts struct {
    status        string              // canonical, already validated
    guids         []string            // positional + stdin, merged
    filters       []map[string]string // 0 or 1 map
    searchText    string
    falsePositive bool
    commandName   string              // "incidents.set-status" | "incidents.resolve"
}

func runStatusChange(cmd *cobra.Command, cli *apiclient.Client, o statusChangeOpts) error
```

## Selection semantics

- GUIDs are merged from positional args **and** `--stdin`, enabling pipelines:
  `incidents list ... -o json | jq -r '.items[].guid' | incidents set-status --status Dismissed --stdin`.
- `--filter key=value` (repeatable) → one `innerFilters` map, e.g.
  `--filter severity=Low --filter clusterName=prod` →
  `[{"severity":"Low","clusterName":"prod"}]`. Comma values pass through untouched
  (backend treats `Low,Medium` as OR).
- `--search` → `?searchText=...` query parameter (the handler reads it from the URL,
  not the body).
- **At least one** of {GUIDs, filters, search} is required; otherwise `CodeBadInput`.
  Combinations are allowed (the backend accepts arrays and filters together).

Request shape:

```text
POST /runtime/incidents/changeStatus?searchText=<search>
{
  "status": "Dismissed",
  "incidentsGuids": [...],
  "innerFilters": [{...}],
  "markedAsFalsePositive": false
}
```

## Status validation

Client-side check against the four `kdr` constants. Input is case-insensitive and
normalized to the canonical capitalized form (`dismissed` → `Dismissed`). An empty or
unknown value returns `CodeBadInput` listing the valid set, so we never send an
invalid status to the server.

## Safety & error handling

Reuses the existing `safety.Wrap` scaffolding already present in `resolve.go`:

- `--dry-run` prints the previewed request (method/url/body) without calling the
  server — the blast-radius check for filter-based bulk.
- `--yes` skips confirmation; otherwise prompt on a TTY.
- Audit log uses command name `incidents.set-status`, with `ArgsLog` capturing the
  selection (guid count / filters / search) and the target status.
- No pre-flight count (decision: keep one code path, zero extra calls).

Errors:

- No selection → `CodeBadInput`.
- Invalid/empty `--status` → `CodeBadInput` listing valid values.
- Server `>= 400` → `CodeServer` with trimmed body + `x-request-id` (identical to
  `resolve`).

## Testing (TDD — tests first)

New `setstatus_test.go`:

- `set-status --status Dismissed i1 i2 --dry-run` → body has `status:"Dismissed"` and
  both GUIDs; server not hit.
- `--filter severity=Low --filter clusterName=prod` → single `innerFilters` map with
  both keys.
- `--search nginx` → request URL carries `?searchText=nginx`.
- `--stdin` → GUIDs read from stdin and merged with positional args.
- Invalid status (`--status Bogus`) and no-selection → `CodeBadInput`, no server hit.
- `--yes --status Dismissed i1` → posts and reports `changed:true`.

Regression: existing `resolve_test.go` (dry-run + `--yes`) stays green **unchanged**,
proving the alias refactor preserved behavior.

## Docs & skill metadata

- `skill.go`: include `Dismissed` in the status field notes (canonical capitalized
  values); add recipes for **bulk dismiss via filter** and **dismiss a GUID list via
  stdin**.
- `types.go`: update the `attributes.incidentStatus` cheatsheet text to list all four
  canonical statuses.
- Add a feature doc under `docs/features/` per the repo docs-gate (required by the
  SessionStart hook on commit).
