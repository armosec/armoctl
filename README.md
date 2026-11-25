# armoctl

CLI tool for instrumenting ECS task definitions with the ARMO runtime security agent.

## 🤖 Use from Claude Code, Gemini CLI, or GitHub Copilot CLI

armoctl ships as a Claude Code plugin (also installable in GitHub Copilot CLI, and as a Gemini CLI extension) so AI assistants can drive the ARMO security platform directly: list incidents, triage CVEs, manage exception policies, generate network policies, and more.

### Claude Code

```
/plugin marketplace add armosec/armoctl
/plugin install armoctl@armosec
```

The first time a session starts, the plugin checks for the `armoctl` binary on `PATH` and runs the official installer if it's missing. After that, whenever a released binary's version differs from the plugin's, the SessionStart hook reinstalls the plugin's version in place, so the binary always matches the skills it ships with. A locally built binary is left alone.

### Gemini CLI

```
gemini extension install https://github.com/armosec/armoctl
```

Gemini CLI first tries to fetch a release-asset bundle and fails with a 404, then offers a `git clone` fallback — answer **Y** to that prompt and accept the one-time review of hooks/skills. The extension loads the same skills as the Claude plugin and the same SessionStart hook installs/updates the `armoctl` binary on the first session. No need to install the binary manually first.

### GitHub Copilot CLI

```
copilot plugin marketplace add armosec/armoctl
copilot plugin install armoctl@armosec
```

Copilot CLI reads the same `.claude-plugin/` manifests, skills, and `hooks/hooks.json` as Claude Code — there is no Copilot-specific manifest. The first time a session starts, the SessionStart hook installs the `armoctl` binary if it's missing, then keeps it on the plugin's version. Check with `copilot plugin list`, or `/skills list` inside a session.

### What's in the plugin

- A root `armoctl` skill covering setup, the JSON output contract (`--full` / `--fields` / `--query`), the mutation safety contract (`--dry-run` / `--yes`), and error semantics.
- 13 per-cluster skills (`armoctl-incidents`, `armoctl-vulns`, `armoctl-posture`, `armoctl-risks`, `armoctl-attack-chains`, `armoctl-inventory`, `armoctl-network-policies`, `armoctl-seccomp`, `armoctl-runtime-rules`, `armoctl-runtime-policies`, `armoctl-integrations`, `armoctl-cloud-accounts`, `armoctl-repo-posture`) auto-loaded by description match when the user's task touches that cluster.
- A SessionStart hook that ensures the binary is present and version-matched. One `hooks/hooks.json` serves all three hosts: Claude Code substitutes `${CLAUDE_PLUGIN_ROOT}`, Gemini CLI substitutes `${extensionPath}`, and Copilot CLI exports `CLAUDE_PLUGIN_ROOT` into the hook's environment. The hook leaves non-release builds (`dev`, `v0.0.13-dirty`) alone.

### Configure once

You'll need two credentials:

- **Customer GUID** — find it in the ARMO Platform UI, top-right account dropdown.
- **Access key** — generate one at <https://cloud.armosec.io/settings/workspace/agent-access-keys> (or <https://cloud.us.armosec.io/settings/workspace/agent-access-keys> for US tenants).

Pick whichever path fits the situation:

**Through the AI assistant (recommended for chat-driven setup).** Paste both credentials into the conversation and let the assistant run:

```bash
echo "<ACCESS_KEY>" | armoctl configure \
    --customer-guid "<CUSTOMER_GUID>" \
    --access-key-stdin
```

Reading the key from stdin keeps it out of shell history and `ps` listings. The command validates against the ARMO API and exits non-zero if the credentials are rejected.

**Interactive on the terminal (no AI involvement):**

```bash
armoctl configure
```

Walks you through Customer GUID, Access Key, and API URL via a TUI. Saved to `~/.armoctl/config.yaml`.

**Environment variables (CI / containers / one-off shells):**

```bash
export ARMO_CUSTOMER_GUID=...
export ARMO_ACCESS_KEY=...
export ARMO_API_BASE_URL=api.armosec.io   # default; override for staging
```

Once configured, the agent can run any read-only command directly. Mutations require `--dry-run` for the preview and `--yes` to commit (or a confirmation prompt on a TTY). The skills tell the agent to show you the dry-run and wait for your explicit confirmation before it runs `--yes`.

## 📦 Install

```bash
curl -fsSL https://package-distribution.armosec.io/armoctl/install.sh | bash
```

## 🔨 Build

```bash
make armoctl
```

## 🔑 Authentication

Most armoctl commands talk to the ARMO platform API and require credentials — the API-backed skills (incidents, vulnerabilities, posture, risks, attack chains, runtime rules/policies, network policies, seccomp, integrations, cloud accounts, repository posture, inventory) authenticate with `customer-guid` and `access-key`. Local helpers like the per-cluster `fields` cheatsheets work offline.

The ECS commands (`ecs patch`, `ecs instrument`) also need credentials: they ask the ARMO API which agent image to use for the AWS region of your AWS config. With an explicit `--agent-image`, previewing a patched task definition works offline; only `--register` and `--deploy` then need credentials.

Configure credentials once with `armoctl configure` (see [Configure once](#configure-once) for the chat-driven, interactive, and env-var paths). They persist to `~/.armoctl/config.yaml` (mode 0600). Env vars (`ARMO_CUSTOMER_GUID`, `ARMO_ACCESS_KEY`, `ARMO_API_BASE_URL`, and `ARMO_API_URL` for the legacy ECS/version-check host) override the config file for the current shell.

## 📋 Commands

### `ecs patch` — Patch a task definition file

Takes a task definition JSON, injects the ARMO ptrace sidecar, and outputs the patched version.

Run `armoctl ecs operator install -c <cluster-arn>` for the cluster first: it registers the cluster with ARMO and grants your AWS account pull access to ARMO's agent images in that region. The agent image (`<ARMO ECR>/ecs-ptrace-agent:<version>`) comes from the ARMO API for the region of the task-definition ARN you pass, or else of your AWS config (`AWS_REGION` or your profile); see [docs/features/ecs-image-resolution.md](docs/features/ecs-image-resolution.md).

```bash
# From a file
armoctl ecs patch task-definition.json

# From stdin
cat task-definition.json | armoctl ecs patch -

# From an ARN (fetches from AWS)
armoctl ecs patch arn:aws:ecs:us-east-1:123456789:task-definition/my-task:1

# Patch only specific containers
armoctl ecs patch --container web --container api task-definition.json

# Patch and register with AWS
armoctl ecs patch --register task-definition.json

# Use a custom agent image
armoctl ecs patch --agent-image my-registry.example.com/ecs-ptrace-agent:v1.0.0 task-definition.json
```

### `ecs instrument` — Instrument a live ECS service

Fetches the current task definition from a running service, patches it, and optionally deploys the update.

```bash
# Preview the patched output
armoctl ecs instrument -c my-cluster -s my-service

# Patch, register, and deploy
armoctl ecs instrument -c my-cluster -s my-service --deploy
```

### `version` — Display version info

```bash
armoctl version
```

## ⚙️ What the patcher does

When you patch a task definition, armoctl:

1. Adds a `sidecar-ptrace` container running the ARMO agent with `SYS_PTRACE` capability
2. Prepends the ptrace shim to each target container's `entryPoint`, so the container starts through the shim with its original argv. When the task definition does not set `entryPoint` (or sets neither `entryPoint` nor `command`), armoctl reads the image ENTRYPOINT/CMD from its registry
3. Sets `pidMode` to `"task"` for cross-container visibility
4. Adds `shared-data` and `profiles-data` volumes
5. Optionally adds a `volume-fixer` init container that prepares shared volumes (`--volume-fixer`)

If armoctl cannot read an image's ENTRYPOINT/CMD (registry unreachable, no credentials, no `linux` image for the task's CPU architecture), patching fails and the error names each affected container. Fix it by setting `entryPoint` and `command` explicitly for that container, or leave the container out of the targets with `--container`. Patching never skips a target container silently.

## 🚩 Flags reference

### Credential flags (`armoctl configure` only)

| Flag | Env var | Description |
|------|---------|-------------|
| `--customer-guid` | `ARMO_CUSTOMER_GUID` | ARMO customer GUID. |
| `--access-key` | `ARMO_ACCESS_KEY` | ARMO API access key. Avoid in shell history; prefer `--access-key-stdin`. |
| `--access-key-stdin` | | Read the access key from stdin. Recommended for scripts and AI agents. |
| `--api-base-url` | `ARMO_API_BASE_URL` | Agent-bridge API host used by every skill cluster (default: `api.armosec.io`). |
| `--api-url` | `ARMO_API_URL` | Legacy backend host passed to the ECS operator and used by version-check (default: `cloud.armosec.io`). |

Env vars are also honored by the rest of armoctl at runtime, regardless of whether `armoctl configure` was used.

### ECS flags

| Flag | Description |
|------|-------------|
| `--agent-image` | Agent sidecar image (default: the image the ARMO API returns for your AWS region; skips the lookup when set). |
| `--container` | Container names to patch (repeatable; default: all). |
| `--volume-fixer` | Include a volume-fixer init container to chmod the shared volume. |
| `--register` | Register the patched task definition with AWS (`patch` only, requires credentials). |
| `--deploy` | Register and deploy to the live service (`instrument` only, requires credentials). |
| `-c`, `--cluster` | ECS cluster name or ARN (`instrument` only). |
| `-s`, `--service` | ECS service name or ARN (`instrument` only). |

### Global flags

Inherited from the root command. The output/query/pagination flags apply to API-backed skill commands that go through the shared response renderer; local helpers like `configure`, `ecs patch`, and the per-cluster `fields` cheatsheets write their output directly and ignore them.

| Flag | Description |
|------|-------------|
| `--output` | Output format: `json` (default), `yaml`, `ndjson`, `table`, `csv`. |
| `--query` | gojq expression applied to the response. |
| `--fields` | Comma-separated dotted paths to keep. |
| `--full` | Disable summary projection; return raw response. |
| `--limit` | Max items to fetch when auto-paging (default 500, 0 = no cap). |
| `--page` / `--page-size` | Explicit pagination. |
| `--dry-run` | Build the request and print the would-be payload without sending it. Recommended preview before any mutation. |
| `--yes` | Skip the confirmation prompt for mutations (required when stdin is not a TTY). |
| `--debug` | Enable debug mode. |
