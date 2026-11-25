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
  - ecs/operator/upgrade.go
  - ecs/operator/cloudformation.go
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
| `ecs operator install -c <arn>`, `ecs operator upgrade -c <arn>` | the cluster ARN's region | `--operator-image` |
| `ecs patch <task-def ARN>`, `ecs instrument -c <cluster ARN>` | the ARN's region | `--agent-image` |
| `ecs patch <file or ->`, `ecs instrument -c <cluster name>` | the AWS config region (`AWS_REGION` or profile) | `--agent-image` |

An explicit flag skips the image lookup. `--agent-image` then makes no backend call at all (offline preview works); `--operator-image` still leaves the cluster registration below in place, since that is what grants pull access. If the lookup
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

Registration is processed asynchronously, so after a successful POST the GET
is retried every 3 s for up to 60 s while it answers "not found". A POST
auth error stops before the GET. Any other failed POST is ignored when the GET
succeeds (the cluster is already registered) and reported at once when the GET
answers "not found".

Run `operator install` before `ecs patch`: until the account is registered for
the region, Fargate tasks cannot pull the sidecar image.

## Upgrading the operator

`ecs operator upgrade -c <arn>` resolves the operator image the same way and
updates the stack's `EcsOperatorImage` parameter, keeping the template and every
other parameter. CloudFormation then rolls the service. If the stack already
runs that image, nothing changes. (Images are pinned to a version, so a plain
redeploy would re-pull the same tag.)
