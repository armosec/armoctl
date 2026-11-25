---
type: feature
status: active
owner: armosec
scope: repo
related_code:
  - internal/sensors/sensors.go
  - ecs/ecs.go
  - ecs/aws.go
  - ecs/operator/register.go
  - ecs/operator/install.go
---

# ECS image resolution and cluster registration

armoctl does not hardcode ARMO's ECR account, region or image tags. The ARMO
API is the source of truth for both, because the ECR account differs per
environment (prod and stage use different AWS accounts).

## Where images come from

`GET /api/v1/sensors/version?region=<region>` returns the pinned versions and,
for that region, the tagless ECR repositories:

```json
{"ptraceAgent":"v0.0.613","ecsOperator":"v0.0.32",
 "images":{"ptraceAgent":"<acct>.dkr.ecr.<region>.amazonaws.com/ecs-ptrace-agent",
           "ecsOperator":"<acct>.dkr.ecr.<region>.amazonaws.com/ecs-operator"}}
```

armoctl builds `repository:version` (`internal/sensors`).

| Command | Region used | Override |
|---|---|---|
| `ecs operator install -c <arn>` | the cluster ARN's region | `--operator-image` |
| `ecs patch`, `ecs instrument` | the AWS config region (`AWS_REGION` or profile) | `--agent-image` |

An explicit flag skips the lookup entirely (works offline). If the lookup
fails, or the API returns no image for the region, the command stops with an
error naming the flag. There is no fallback to a guessed image.

## Cluster registration (operator install)

The images are private. `ecs operator install` onboards the cluster the same
way the ARMO UI does, before creating the operator stack:

1. `POST /api/v1/accounts/ecs {"clusterARN": ...}` registers the cluster, so it
   appears in the UI.
2. `GET /api/v1/accounts/ecs?clusterARN=...` adds the cluster's AWS account to
   the pull policy of ARMO's ECS repositories in that region (operator, EC2
   agent, ptrace sidecar).

Registration is processed asynchronously, so the GET is retried every 3 s for
up to 60 s while it answers "not found". A failed POST (for example, the
cluster is already registered) is ignored when the GET succeeds. Auth errors
stop immediately.

Run `operator install` before `ecs patch`: until the account is registered for
the region, Fargate tasks cannot pull the sidecar image.
