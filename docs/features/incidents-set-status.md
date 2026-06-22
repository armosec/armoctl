# incidents set-status

`armoctl incidents set-status` changes the status of one or more runtime incidents.
It calls `POST /runtime/incidents/changeStatus`, which the backend processes
asynchronously (status updates are published to Pulsar; `searchText` is resolved to
incident GUIDs and cluster names are enriched server-side).

## Usage

```
armoctl incidents set-status [guid...] --status <Open|Investigating|Dismissed|Resolved> [flags]
```

| Flag | Description |
|------|-------------|
| `--status` | Required. Target status (case-insensitive; normalized to canonical form). |
| `--filter key=value` | Repeatable. Builds one `innerFilters` map (AND across filters). |
| `--search` | Free-text; sent as the `searchText` query parameter. |
| `--stdin` | Read additional GUIDs from stdin (whitespace/newline separated). |
| `--false-positive` | Sets `markedAsFalsePositive`. |
| `--dry-run` | Preview the request without sending it. |
| `--yes` | Confirm the mutation in non-interactive mode. |

At least one of GUIDs, `--filter`, or `--search` is required.

## Selection examples

```
# Specific incidents
armoctl incidents set-status i1 i2 --status Dismissed

# Everything matching a filter (one async call)
armoctl incidents set-status --status Dismissed --filter severity=Low --filter clusterName=prod

# Pipe a GUID list from list output
armoctl incidents list --severity Low -o json | jq -r '.items[].guid' | \
  armoctl incidents set-status --status Dismissed --stdin --yes
```

## Safety

`set-status` is a mutation: use `--dry-run` to preview the request without sending it,
and `--yes` to confirm in non-interactive mode. Actions are written to the audit log.

## Relationship to `resolve`

`armoctl incidents resolve <guid>` is a thin alias that sets status to `Resolved`
(with optional `--false-positive`). Both commands share one internal request path.

