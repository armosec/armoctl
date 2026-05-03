---
name: armoctl-runtime-rules
description: Runtime detection rules — CRUD on the per-rule policy surface (the ARMO equivalent of a Falco rule). Use to add, modify, or evaluate runtime rules.
---

# armoctl-runtime-rules

A rule is the smallest unit of runtime detection: 'fire when X happens.' Rules are bundled into runtime policies (next cluster). Custom rules have ruleType 'Custom'; ARMO-managed rules have ruleType 'Managed' and cannot be deleted.

## Commands

- `create` — Create a runtime rule
  - Flags:
    - `--description` (string) Rule description
    - `--name` (string) Rule name (required)
    - `--policy-types` (stringSlice) Policy types (e.g. ADR, CDR)
    - `--rule` (string) Rule expression as JSON string (or use --rule-file)
    - `--rule-file` (string) Path to JSON file containing the rule
- `delete [ruleGUID]` — Delete a runtime rule
- `evaluate` — Evaluate a runtime rule against input data
  - Flags:
    - `--input` (string) Input data as JSON string (or use --input-file)
    - `--input-file` (string) Path to JSON file containing the input data
    - `--rule` (string) Rule expression as JSON string (or use --rule-file)
    - `--rule-file` (string) Path to JSON file containing the rule
- `fields` — Print the runtime-rules resource cheatsheet
- `get [ruleGUID]` — Get a single runtime rule by GUID
- `list` — List runtime rules
  - Flags:
    - `--name` (string) Filter by rule name
- `update` — Update a runtime rule
  - Flags:
    - `--description` (string) Rule description
    - `--guid` (string) Rule GUID (required)
    - `--name` (string) Rule name
    - `--policy-types` (stringSlice) Policy types (e.g. ADR, CDR)
    - `--rule` (string) Rule expression as JSON string (or use --rule-file)
    - `--rule-file` (string) Path to JSON file containing the rule

## Resource fields

### runtime-rules

| Field | Description |
|---|---|
| `guid` | Stable rule ID. |
| `name` | Rule name. |
| `description` | Human-readable description. |
| `ruleType` | Managed \| Custom. |
| `createdBy` | Author. |
| `creationTime` | RFC3339 creation. |
| `policyTypes` | ADR \| CDR etc. |
| `rule` | Full rule expression (in --full). |

## Field semantics

**`policyTypes`** — Policy type categories this rule belongs to (e.g. ADR, CDR). Used to group rules into policy bundles.

**`ruleType`** — Managed | Custom. Managed rules are maintained by ARMO and cannot be deleted. Custom rules are user-created and fully mutable.

## Recipes

### Create a rule from a JSON file

```
armoctl runtime-rules create --name 'my-rule' --rule-file rule.json --dry-run
```

