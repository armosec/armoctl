---
name: armoctl-cloud-accounts
description: Cloud account onboarding — list/connect/disconnect ECS accounts. Use to see which AWS accounts ARMO is monitoring or to onboard a new one.
---

# armoctl-cloud-accounts

Cloud accounts is the AWS-side onboarding surface. Today it covers ECS account connection state; future cloud surfaces (EKS, GCP) will land here.

## Commands

- `ecs` — ECS cluster connections
- `ecs connect [cluster-arn]` — Get the CloudFormation install link for an ECS cluster
- `ecs disconnect [cluster-arn]` — Disconnect an ECS cluster from ARMO
- `ecs list` — List ECS cluster connections
- `fields` — Print the ECS cluster resource cheatsheet

## Mutations

Any command that writes (create, update, delete, status changes, …) follows the safety contract of the `armoctl` skill:

1. Run it with `--dry-run` and show the user what will change.
2. Ask the user to confirm and wait for an explicit "yes". The original request is not confirmation.
3. After the user confirms, re-run the identical command with `--yes` in place of `--dry-run`. If any argument changes, dry-run and confirm again.

This holds even when the host auto-approves `armoctl` commands.

## Resource fields

### ecs

| Field | Description |
|---|---|
| `clusterARN` | AWS ECS cluster ARN. |
| `name` | Cluster name. |
| `region` | AWS region. |
| `accountID` | AWS account ID. |
| `status` | Connection status. |
| `lastSeen` | RFC3339 last-seen time. |

## Field semantics

**`status`** — Connection status: connected / pending / failed. 'pending' means CloudFormation rollout is still in progress.

## Recipes

### List ECS accounts

```
armoctl cloud-accounts ecs list
```

