# armoctl

CLI tool for instrumenting ECS task definitions with the ARMO runtime security agent.

## 🤖 Use from Claude Code or Gemini CLI

armoctl ships as a Claude Code plugin (and Gemini CLI extension) so AI assistants can drive the ARMO security platform directly: list incidents, triage CVEs, manage exception policies, generate network policies, and more.

### Claude Code

```
/plugin marketplace add armosec/armoctl
/plugin install armoctl@armosec
```

The first time a session starts, the plugin checks for the `armoctl` binary on `PATH` and runs the official installer if it's missing. After that, the SessionStart hook keeps the binary on the same version as the plugin (running `armoctl update` whenever they drift).

### Gemini CLI

Add this repo as an extension. The Gemini extension loads the same skills as the Claude plugin from `skills/`. Install the binary first (see the next section).

### What's in the plugin

- A root `armoctl` skill covering setup, the JSON output contract (`--full` / `--fields` / `--query`), the mutation safety contract (`--dry-run` / `--yes`), and error semantics.
- 13 per-cluster skills (`armoctl-incidents`, `armoctl-vulns`, `armoctl-posture`, `armoctl-risks`, `armoctl-attack-chains`, `armoctl-inventory`, `armoctl-network-policies`, `armoctl-seccomp`, `armoctl-runtime-rules`, `armoctl-runtime-policies`, `armoctl-integrations`, `armoctl-cloud-accounts`, `armoctl-repo-posture`) auto-loaded by description match when the user's task touches that cluster.
- A SessionStart hook that ensures the binary is present and version-matched.

### Configure once

You'll need two credentials:

- **Customer GUID** — find it in the ARMO Platform UI, top-right account dropdown.
- **Access key** — generate one at <https://cloud.armosec.io/settings/workspace/agent-access-keys> (or <https://cloud.us.armosec.io/settings/workspace/agent-access-keys> for US tenants).

Then either:

```bash
armoctl configure   # interactive — saves to ~/.armoctl/config.yaml
# or via env vars (preferred for headless agents):
export ARMO_CUSTOMER_GUID=...
export ARMO_ACCESS_KEY=...
```

Once configured, the agent can run any read-only command directly. Mutations require `--dry-run` for the preview and `--yes` to commit (or a confirmation prompt on a TTY).

## 📦 Install

```bash
curl -fsSL https://package-distribution.armosec.io/armoctl/install.sh | bash
```

## 🔨 Build

```bash
make armoctl
```

## 🔑 Authentication

Credentials are only required when using `--register` / `--deploy` to push changes to AWS. Preview/patch operations work without authentication.

Pass credentials via flags, environment variables, or config file:

```bash
# Flags
armoctl --customer-guid <GUID> --access-key <KEY> ecs patch --register ...

# Environment variables
export ARMO_CUSTOMER_GUID=<GUID>
export ARMO_ACCESS_KEY=<KEY>
armoctl ecs patch --register ...

# Config file (~/.armoctl/config.yaml)
customer-guid: <GUID>
access-key: <KEY>
```

## 📋 Commands

### `ecs patch` — Patch a task definition file

Takes a task definition JSON, injects the ARMO ptrace sidecar, and outputs the patched version.

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
armoctl ecs patch --agent-image 015253967648.dkr.ecr.eu-north-1.amazonaws.com/ecs-ptrace-agent:v1.0.0 task-definition.json
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
2. Wraps each target container's command to launch through the ptrace shim
3. Sets `pidMode` to `"task"` for cross-container visibility
4. Adds `shared-data` and `profiles-data` volumes
5. Optionally adds a `volume-fixer` init container that prepares shared volumes (`--volume-fixer`)

Containers without a `command` (relying on image ENTRYPOINT/CMD) are instrumented via dependency and volume mounts but their entrypoint is not modified.

## 🚩 Flags reference

| Flag | Env var | Description |
|------|---------|-------------|
| `--customer-guid` | `ARMO_CUSTOMER_GUID` | ARMO customer GUID |
| `--access-key` | `ARMO_ACCESS_KEY` | ARMO API access key |
| `--api-url` | `ARMO_API_URL` | ARMO platform URL (default: `cloud.armosec.io`) |
| `--agent-image` | | Agent sidecar image (default: `015253967648.dkr.ecr.eu-north-1.amazonaws.com/ecs-ptrace-agent:latest`) |
| `--container` | | Container names to patch (repeatable; default: all) |
| `--volume-fixer` | | Include a volume-fixer init container to chmod the shared volume |
| `--register` | | Register the patched task definition with AWS (`patch` only, requires credentials) |
| `--deploy` | | Register and deploy to the live service (`instrument` only, requires credentials) |
| `-c`, `--cluster` | | ECS cluster name or ARN (`instrument` only) |
| `-s`, `--service` | | ECS service name or ARN (`instrument` only) |
| `--debug` | | Enable debug mode |
