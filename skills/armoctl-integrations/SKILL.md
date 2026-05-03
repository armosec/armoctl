---
name: armoctl-integrations
description: Outbound integrations — alert channels (Slack/email/webhook), SIEM forwarders, Jira ticket creation. Use to wire ARMO into external workflows.
---

# armoctl-integrations

Integrations is where ARMO emits, not consumes. Alert channels deliver events; SIEM forwarders ship logs; Jira lets the agent open tickets directly.

## Commands

- `alert-channels` — Alert channel integrations
- `alert-channels create [guid]` — Create an alert channel
  - Flags:
    - `--config-file` (string) JSON file with channel-specific config (optional)
    - `--guid` (string) Channel GUID (optional if provided as positional)
    - `--type` (string) Channel type (slack|email|webhook) (required)
- `fields` — Print the integrations resource cheatsheet
- `jira` — Jira integration commands
- `jira create-ticket` — Create a Jira issue (ticket)
  - Flags:
    - `--description` (string) Issue description (optional)
    - `--field` (stringSlice) Extra fields as key=value (repeatable, optional)
    - `--issue-type` (string) Issue type name (required)
    - `--project` (string) Project key (required)
    - `--summary` (string) Issue summary (required)
- `jira fields` — List Jira fields available for an issue type
  - Flags:
    - `--issue-type` (string) Issue type ID (required)
    - `--project` (string) Project key (required)
- `jira issue-types` — List Jira issue types
  - Flags:
    - `--project` (string) Filter by project key
- `jira projects` — List Jira projects
- `siem` — SIEM integrations
- `siem create [provider]` — Create SIEM integration for the given provider
  - Flags:
    - `--config-file` (string) JSON file with provider-specific config (required)
- `unlink [guid]` — Unlink an integration by GUID

## Resource fields

### jira-issue-types

| Field | Description |
|---|---|
| `id` | Issue type ID. |
| `name` | Issue type name (Bug, Task, Epic, ...). |
| `description` | Description. |
| `subtask` | Whether this is a subtask type. |

### jira-projects

| Field | Description |
|---|---|
| `key` | Project key (e.g. SEC). |
| `name` | Human-readable project name. |
| `id` | Numeric Jira project ID. |
| `projectTypeKey` | software \| service_desk \| business. |
| `lead` | Project lead identifier. |

## Field semantics

**`projectTypeKey`** — Jira project type: software | service_desk | business. Determines which issue types and workflows are available in the project.

## Recipes

### Create a Jira ticket from an incident

```
armoctl integrations jira create-ticket --project <key> --issue-type Bug --summary 'Incident <guid>'
```

