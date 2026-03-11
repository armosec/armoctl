# armoctl

CLI tool for instrumenting ECS task definitions with the ARMO runtime security agent.

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
