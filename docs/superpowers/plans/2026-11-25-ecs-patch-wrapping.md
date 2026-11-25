# ecs patch wrapping Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Every target container in `armoctl ecs patch` starts through the ptrace shim with its original argv (SUB-10274, SUB-10273), plus the armoctl part of SUB-10278.

**Architecture:** The pure `ecs/patcher` package moves the shim into `entryPoint` and asks an injected `ImageConfigResolver` for the image ENTRYPOINT/CMD when the task def lacks them. A new `ecs/imageconfig` package implements that resolver with go-containerregistry (ECR auth via AWS SDK, Docker keychain, anonymous). The CLI wires it in and prints the registered revision ARN.

**Tech Stack:** Go 1.25, aws-sdk-go-v2 (ecs, ecr), github.com/google/go-containerregistry, cobra.

**Spec:** `docs/superpowers/specs/2026-11-25-ecs-patch-wrapping-design.md`

## Global Constraints

- Shim path stays `/tmp/shared/ptrace-shim` (`shimPath`).
- Error copy, verbatim, per failing container:
  `container "<name>": cannot read ENTRYPOINT/CMD of image "<image>": <cause>. Set "entryPoint" and "command" explicitly for this container in the task definition, or leave it out of the targets with --container.`
- All failing containers are reported together (`errors.Join`), and on any error the task def is not mutated.
- No registry lookup for containers that set `entryPoint`.
- Platform: `linux` + `runtimePlatform.cpuArchitecture` (`ARM64` → `arm64`, anything else/unset → `amd64`).
- Repo has `docs/`: doc updates ship in the same commit as the behavior (docs-gate hook).
- Never commit armoctl access keys or real patched task-def output.

## Review Focus

1. Container with `entryPoint` set and empty `command`: must stay `[shim]+entryPoint`, `command` empty, no lookup (Task 1 test `entrypoint_only`).
2. Image whose config has ENTRYPOINT but no CMD (and vice versa): argv must be exactly the non-empty part (Task 1 tests `image_entrypoint_only`, `image_cmd_only`).
3. Resolver error for one of two containers: exactly one error, naming that container; td unchanged (Task 1 test `TestPatchResolverErrorLeavesTaskDefUntouched`).
4. Multi-arch index with no matching platform: clear error, not a panic (Task 2 test `TestResolveNoMatchingPlatform`).
5. Re-patching a container whose `entryPoint[0]` is already the shim (no sidecar present, e.g. hand-edited): left unchanged, no double shim (Task 1 test `already_shimmed`).

---

### Task 1: Patcher wraps via entryPoint using a resolver

**Files:**
- Modify: `ecs/patcher/types.go` (PatchOptions, new types)
- Modify: `ecs/patcher/patcher.go` (Patch signature, wrapping, health check)
- Modify: `ecs/patcher/patcher_test.go`
- Modify: `ecs/ecs.go:92-103` (patchAndPrint passes ctx; compile only)

**Interfaces:**
- Produces:
  - `type ImageConfig struct { Entrypoint []string; Cmd []string }`
  - `type ImageConfigResolver interface { Resolve(ctx context.Context, image string, arch ecstypes.CPUArchitecture) (ImageConfig, error) }`
  - `PatchOptions.Resolver ImageConfigResolver` (nil allowed; a needed lookup with nil resolver is an error with the standard copy, cause `no image resolver configured`)
  - `func Patch(ctx context.Context, td *TaskDefinition, opts PatchOptions, sidecar SidecarConfig) error`
  - `wrapCommand` removed.

- [ ] **Step 1: Write failing tests** in `patcher_test.go`. Add a `fakeResolver struct { configs map[string]ImageConfig; errs map[string]error; calls []string }`. Replace `TestWrapCommand` and `TestPatchEmptyCommand` with table test `TestPatchWrapping` (one container, image `img`, assert `EntryPoint`, `Command`, and `resolver.calls`):

| case | entryPoint | command | image config | want entryPoint | want command | lookup |
|---|---|---|---|---|---|---|
| entrypoint_and_command | `[/ep.sh]` | `[run]` | – | `[shim /ep.sh]` | `[run]` | no |
| entrypoint_only | `[/ep.sh]` | nil | – | `[shim /ep.sh]` | nil | no |
| command_only | nil | `[nginx -g daemon off;]` | EP `[/docker-entrypoint.sh]` | `[shim /docker-entrypoint.sh]` | `[nginx -g daemon off;]` | yes |
| image_defaults | nil | nil | EP `[/docker-entrypoint.sh]`, Cmd `[nginx -g daemon off;]` | `[shim /docker-entrypoint.sh]` | `[nginx -g daemon off;]` | yes |
| image_entrypoint_only | nil | nil | EP `[/app]` | `[shim /app]` | nil | yes |
| image_cmd_only | nil | nil | Cmd `[python app.py]` | `[shim]` | `[python app.py]` | yes |
| already_shimmed | `[shim /ep.sh]` | `[run]` | – | `[shim /ep.sh]` | `[run]` | no |

  Plus:
  - `TestPatchEmptyArgvError`: no entryPoint/command, image config empty → error contains `container "app"` and `cannot read ENTRYPOINT/CMD of image "img"` and cause `image defines no ENTRYPOINT or CMD`.
  - `TestPatchResolverErrorLeavesTaskDefUntouched`: two containers needing lookup, resolver errors for `b` only → error contains the full Global Constraints copy for `b`, not `a`; td deep-equals a copy taken before Patch (no sidecar, no volumes).
  - `TestPatchNilResolverNeedsLookup`: command-only container, `Resolver: nil` → error with cause `no image resolver configured`.
  - `TestPatchResolverArch`: `RuntimePlatform.CpuArchitecture = ARM64` → fake sees `ARM64`; unset → fake sees `""` (mapping happens in Task 2).
  - In `TestPatchMinimalTaskDef` (or new test) assert sidecar health check equals `["CMD", "/usr/bin/ptrace-agent", "--health", "--shim"]`.
  - Update every existing `Patch(...)` call to `Patch(context.Background(), ...)` and give containers that need lookups a fake resolver.

- [ ] **Step 2: Run** `go test ./ecs/patcher/` — expect compile failure / FAIL.

- [ ] **Step 3: Implement.** In `Patch`, after `resolveTargets` and before any mutation, compute a `map[string]wrapped{entryPoint, command}` for each target via `wrapContainer(ctx, c, arch, resolver) ([]string, []string, error)`; collect errors with `errors.Join`; return on error. Then mutate as before, assigning the computed `EntryPoint`/`Command`. Fix the health-check arg to `"--shim"`.

- [ ] **Step 4: Run** `go test ./ecs/... && go vet ./...` — PASS.

- [ ] **Step 5: Update docs** in the same commit: README "What the patcher does" (step 2 becomes "Prepends the ptrace shim to each target container's `entryPoint`, reading the image ENTRYPOINT/CMD from the registry when the task definition does not set them"; delete the line saying image-default containers are not modified; add the error behavior). Add `docs/features/ecs-patch-wrapping.md` (frontmatter like `ecs-image-resolution.md`, `related_code: ecs/patcher/patcher.go, ecs/imageconfig/resolver.go, ecs/ecs.go`) with the rules table from the spec.

- [ ] **Step 6: Commit** `fix(ecs): wrap containers through entryPoint, cover image defaults (SUB-10274, SUB-10273)`.

### Task 2: Registry-backed resolver

**Files:**
- Create: `ecs/imageconfig/resolver.go`, `ecs/imageconfig/resolver_test.go`
- Modify: `go.mod`, `go.sum` (`github.com/google/go-containerregistry`, `github.com/aws/aws-sdk-go-v2/service/ecr`)

**Interfaces:**
- Consumes: `patcher.ImageConfig`, `patcher.ImageConfigResolver`.
- Produces:
  - `func New(awsCfg *aws.Config) *Resolver` (nil cfg → no ECR auth; Docker keychain + anonymous only)
  - `func (r *Resolver) Resolve(ctx context.Context, image string, arch ecstypes.CPUArchitecture) (patcher.ImageConfig, error)`

- [ ] **Step 1: Write failing tests** using `github.com/google/go-containerregistry/pkg/registry` (in-memory, `httptest.NewServer`) and `random.Image` with `mutate.Config`:
  - `TestResolveSingleImage`: push image with Entrypoint `[/ep]`, Cmd `[run]` → returned as is.
  - `TestResolveMultiArch`: index with amd64 (Cmd `[amd]`) and arm64 (Cmd `[arm]`) → `ARM64` gives `[arm]`; `""` and `X86_64` give `[amd]`.
  - `TestResolveNoMatchingPlatform`: index with only arm64, ask `""` → error mentioning `linux/amd64`.
  - `TestResolveMissingImage`: unknown tag → error (no panic).
  - `TestIsECRRegistry`: `123456789012.dkr.ecr.eu-west-1.amazonaws.com` → region `eu-west-1`, true; `public.ecr.aws`, `docker.io`, `quay.io` → false.
- [ ] **Step 2: Run** `go test ./ecs/imageconfig/` — FAIL.
- [ ] **Step 3: Implement.** `name.ParseReference(image)`; `remote.Image(ref, remote.WithContext(ctx), remote.WithAuthFromKeychain(kc), remote.WithPlatform(v1.Platform{OS: "linux", Architecture: archFor(arch)}))`; return `cfg.Config.Entrypoint` / `cfg.Config.Cmd`. Keychain = `authn.NewMultiKeychain(ecrKeychain, authn.DefaultKeychain)` (falls back to anonymous). `ecrKeychain` matches `<acct>.dkr.ecr.<region>.amazonaws.com[.cn]`, calls `ecr.GetAuthorizationToken` with a client for that region (copy of awsCfg with Region overridden), decodes `user:password`, caches per region. `remote.Image` already rejects a missing platform; wrap errors with `fmt.Errorf("resolving %s for linux/%s: %w", ...)`. Tests use `name.Insecure` via an unexported option for the httptest registry.
- [ ] **Step 4: Run** `go test -race ./ecs/imageconfig/ && go mod tidy && go build ./...` — PASS.
- [ ] **Step 5: Commit** `feat(ecs): resolve image ENTRYPOINT/CMD from the registry`.

### Task 3: CLI wiring and revision ARN

**Files:**
- Modify: `ecs/ecs.go` (`patchOpts`), `ecs/patch.go:52-61`, `ecs/ecs_test.go`, README (ECS credentials paragraph, line ~94)

**Interfaces:**
- Consumes: `imageconfig.New`, `PatchOptions.Resolver`.
- Produces: `func patchOpts(cmd *cobra.Command) patcher.PatchOptions` sets `Resolver: imageconfig.New(cfg)` where `cfg` is `awsconfig.LoadDefaultConfig(cmd.Context())`; on load failure pass `nil` cfg (non-ECR images still resolve).

- [ ] **Step 1: Write failing test** in `ecs/ecs_test.go`: `TestPatchOptsSetsResolver` — `patchOpts(cmd).Resolver != nil`.
- [ ] **Step 2: Run** `go test ./ecs/` — FAIL.
- [ ] **Step 3: Implement** `patchOpts`; in `patch.go` print `Registered task definition: <arn>` using the ARN `registerTaskDef` returns (fall back to family if ARN is empty). Update README line ~94: previewing offline with `--agent-image` now also requires every target container to set `entryPoint`; otherwise patching reads image configs from their registries (ECR via your AWS credentials, others via Docker credentials or anonymously).
- [ ] **Step 4: Run** `go test -race ./... && go vet ./... && make armoctl` — PASS.
- [ ] **Step 5: Manual check:** `./armoctl ecs patch --agent-image example.com/a:1 <tmp nginx taskdef with no command>` (scratchpad file, fake creds) prints `entryPoint: ["/tmp/shared/ptrace-shim","/docker-entrypoint.sh"]`, `command: ["nginx","-g","daemon off;"]`; same with image `does-not-exist.invalid/x:1` prints the Global Constraints error.
- [ ] **Step 6: Commit** `fix(ecs): print registered revision ARN (SUB-10278)`.

### Task 4: Release

- [ ] Open PR via `armosec-shared-rules:open_pr` with keys SUB-10274, SUB-10273, SUB-10278; add the `release` label (merge cuts the next `vX.Y.Z`).
- [ ] After merge: rerun the Fargate harness with the two-container task (nginx on image defaults + probe) and confirm the pass signals in CLAUDE.md.
