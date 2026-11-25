# ecs patch: wrap every target container correctly (SUB-10274, SUB-10273, SUB-10278)

## Why

The documented ECS Fargate install must work end to end. Today `armoctl ecs patch`
crashes containers that set `entryPoint` (SUB-10274) and silently leaves
containers that use the image's default ENTRYPOINT/CMD unmonitored (SUB-10273).
Success: the plain two-container task (nginx on image defaults + a probe with an
explicit command) is patched and monitored with no manual edits.

## Goals

- Every target container starts through the ptrace shim, with the same argv it
  had before patching.
- When armoctl can't work out a container's argv, it fails with an explicit,
  actionable error. It never skips silently.
- SUB-10278 (armoctl part): drop the leading space in the sidecar health-check
  argument (`" --shim"` → `"--shim"`), and print the registered revision ARN in
  `ecs patch --register`.

## Non-goals

- Single-target tracing for multi-container tasks (SUB-10275).
- The rest of SUB-10278 (R1223 version skew, agent version stamping, doc link).

## Approaches

1. **Read the image config from the registry at patch time (chosen).** Covers
   image-default containers without user effort.
2. Never fetch; error out for any container without explicit `entryPoint` and
   `command`. Rejected: the documented nginx case would still need manual edits.

Decision (user): option 1; if the image config is unavailable for any reason,
fail with an explicit error.

## Design

ECS/Docker effective argv rules:
- `entryPoint` set → argv = `entryPoint + command` (image ENTRYPOINT and CMD are
  both ignored).
- `entryPoint` unset → argv = `image ENTRYPOINT + (command or image CMD)`.

New wrapping, per target container (replaces `wrapCommand` on `command`):

| Task def has | Lookup? | Result |
|---|---|---|
| `entryPoint` set | no | `entryPoint = [shim] + entryPoint`; `command` unchanged |
| no `entryPoint`, `command` set | yes (image ENTRYPOINT) | `entryPoint = [shim] + imageEntrypoint`; `command` unchanged |
| neither | yes | `entryPoint = [shim] + imageEntrypoint`; `command = imageCmd` |

- Setting `entryPoint` makes Docker drop the image CMD, which is why the last
  row copies `imageCmd` into `command` explicitly.
- Empty effective argv (no entrypoint and no command anywhere) → error.
- `entryPoint[0]` already equal to the shim path → leave the container as is
  (idempotent).

Image lookup:
- `go-containerregistry` (new dependency) behind an `ImageConfigResolver`
  interface passed in via `PatchOptions`, so `patcher` stays pure and tests use a
  fake.
- Auth: ECR images via the AWS credentials armoctl already loads (ECR
  `GetAuthorizationToken` for the image's registry/region); everything else via
  the default Docker keychain, falling back to anonymous.
- Platform: `linux` + the task's `runtimePlatform.cpuArchitecture` (`ARM64` →
  arm64, otherwise amd64) when the image is a multi-arch index.
- Lookups happen only for containers that need them, so a task def with explicit
  `entryPoint` everywhere still patches offline.

Error (one per failing container, all reported together):

```
container "web": cannot read ENTRYPOINT/CMD of image "nginx:1.27": <cause>.
Set "entryPoint" and "command" explicitly for this container in the task
definition, or leave it out of the targets with --container.
```

Other changes:
- `ecs patch --register` prints `Registered task definition: <revision ARN>`.
- README "What the patcher does" and `docs/features/` updated: the line saying
  image-default containers are not modified goes away; note that patching may
  read image configs from the registry.

## Left to the builder

- Exact package layout for the resolver (e.g. `ecs/imageconfig`).
- Test table in `ecs/patcher/patcher_test.go` covering every row above, the
  error path, idempotency, and the health-check argument.
