---
name: armoctl-incidents
description: ARMO runtime incidents — list active threats, fetch alerts on a single incident, explain an incident's signal, resolve/silence incidents. Use when investigating live runtime alerts or post-mortems.
---

# armoctl-incidents

The incidents cluster is the live runtime-threat surface. An incident is the unit of triage; it bundles many alerts produced from runtime detection rules. Severity is ARMO-policy-adjusted, not raw alert severity. Use 'incidents alerts <guid>' to get the full alert payload behind an incident before resolving it.

## Commands

- `alerts [guid]` — List alerts grouped under one incident
- `explain [guid]` — Aggregate the platform's streaming explanation for an incident
- `fields` — Print the incidents resource cheatsheet
- `list` — List runtime incidents
  - Flags:
    - `--severity` (string) Filter by severity (critical|high|medium|low)
- `resolve [guid]` — Resolve a runtime incident (sets status to Resolved)
  - Flags:
    - `--false-positive` (bool) Mark the incident as a false positive when resolving
- `severities` — Get aggregate incident counts per severity

## Resource fields

### incidents

| Field | Description |
|---|---|
| `guid` | Stable incident ID; primary key for resolve/explain/alerts. |
| `name` | Short rule/incident name (e.g. "Suspicious binary execution"). |
| `kind` | Incident kind/category (e.g. "ThreatDetection"). |
| `attributes.incidentStatus` | Current status: open \| resolved \| investigating. Access with path syntax. |
| `updatedTime` | RFC3339 timestamp of the last status change. |
| `timestamp` | RFC3339 time the incident was first raised. |
| `clusterName` | Kubernetes cluster that reported the incident. |
| `designators.wlid` | Workload ID (ARMO wlid format). Access with path syntax. |
| `cloudMetadata.region` | Cloud region where the workload runs. Access with path syntax. |
| `signature` | Unique fingerprint identifying the rule that fired. |

## Field semantics

**`attributes.incidentStatus`** — Live state machine: open → investigating → resolved. Access with path syntax: .attributes.incidentStatus

**`signature`** — Unique fingerprint identifying the rule that fired. Incidents sharing a signature are the same detection event pattern.

## Recipes

### List Critical incidents (all statuses)

```
armoctl incidents list --severity Critical
```

### Get all alerts for an incident

```
armoctl incidents alerts <incident-guid>
```

