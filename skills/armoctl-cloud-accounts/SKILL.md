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

