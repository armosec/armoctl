---
type: feature
status: active
owner: armosec
scope: repo
related_code:
  - ecs/patcher/patcher.go
  - ecs/patcher/types.go
  - ecs/imageconfig/resolver.go
  - ecs/ecs.go
---

# ECS patch: starting containers through the ptrace shim

The ptrace sidecar traces the process tree whose PID the shim writes to
`/tmp/shared/app.pid`. `armoctl ecs patch` therefore has to make every target
container start through the shim (`/tmp/shared/ptrace-shim`), which writes its
PID, stops itself until the agent attaches, then `execve`s the original argv.

## How the argv is rewritten

ECS runs `entryPoint + command`. Without an `entryPoint` it runs the image
ENTRYPOINT followed by `command`, or by the image CMD when `command` is empty
too. Setting `entryPoint` makes Docker ignore the image CMD.

The patcher always puts the shim first in `entryPoint`:

| Task definition has | Registry lookup | Patched `entryPoint` | Patched `command` |
|---|---|---|---|
| `entryPoint` | no | `[shim] + entryPoint` | unchanged |
| `command` only | yes | `[shim] + image ENTRYPOINT` | unchanged |
| neither | yes | `[shim] + image ENTRYPOINT` | image CMD |

A container whose `entryPoint` already starts with the shim is left as is.

## Reading image configs

`ecs/imageconfig` reads the image config with go-containerregistry, for
`linux` and the task's `runtimePlatform.cpuArchitecture` (`ARM64` → arm64,
otherwise amd64) when the image is multi-arch. Registry auth, in order:

- ECR (`<account>.dkr.ecr.<region>.amazonaws.com`): an ECR authorization token
  from the AWS credentials armoctl already uses.
- Anything else: the Docker credential store (`~/.docker/config.json` and
  helpers), then anonymous.

Lookups happen only for containers that need them; a task definition that sets
`entryPoint` on every target container patches offline.

## Failure

All lookups run before the task definition is changed. If any fails, `patch`
leaves it untouched and returns one error per failing container:

```
container "web": cannot read ENTRYPOINT/CMD of image "nginx:1.27": <cause>.
Set "entryPoint" and "command" explicitly for this container in the task
definition, or leave it out of the targets with --container.
```

History: before this, containers with `entryPoint` crashed (the shim ran as an
argument to the entrypoint, SUB-10274) and containers on image defaults were
silently left untraced (SUB-10273).
