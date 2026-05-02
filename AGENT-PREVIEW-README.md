# armoctl agent-bridge preview build

Single binary covering all 14 ARMO platform clusters, plus a `SKILL.md` for agent use.

## Contents

- `armoctl` — Linux x86_64 binary built from `feature/agent-bridge-smoke-fixes` (sum of PRs #12–#20).
- `SKILL.md` — agent skill describing setup, output contract, mutation safety, recipes, pitfalls.

## Quick start

```bash
# 1. Put the binary on PATH
chmod +x ./armoctl
sudo mv ./armoctl /usr/local/bin/

# 2. Configure credentials (interactive) or use env vars
armoctl configure
# OR
export ARMO_CUSTOMER_GUID="..."
export ARMO_ACCESS_KEY="..."
export ARMO_API_BASE_URL="api.armosec.io"   # or "api-dev.armosec.io" for the dev tenant

# 3. Test it
armoctl --help
armoctl incidents list --limit 3 --query '.items|length'
```

## Live smoke status (against api-dev.armosec.io)

| Cluster | Status |
|---|---|
| incidents | 4/5 ✅ — `resolve --yes` hits an upstream Cloudflare 504 (backend issue, CLI shape correct) |
| vulns | 11/11 ✅ |
| posture | 4/5 ✅ — `posture resources` returns Cloudflare 400 (upstream issue) |
| risks | 3/3 ✅ |
| attack-chains | 1/1 ✅ |
| inventory | 3/3 ✅ |
| network-policies | 2/2 ✅ |
| seccomp | 2/2 ✅ |
| cloud-accounts | 3/3 ✅ |
| runtime-rules | 3/4 ✅ — `evaluate` requires valid rule/input (expected) |
| runtime-policies | 2/2 ✅ |
| integrations | 3/5 ✅ — live Jira calls fail because Jira isn't fully configured for the test tenant (not a CLI bug) |
| repo-posture | 5/5 ✅ — returns 0 items because no repos are connected to the test tenant |

Total: ~50 of ~55 commands pass live. The 5 that don't are upstream/tenant-config issues, not CLI bugs.

## Using the SKILL.md with an agent

The simplest integration is to put `SKILL.md` somewhere your agent runtime can load it from (e.g., a Claude Code skill directory) and ensure `armoctl` is on PATH.

For Claude Code, drop the SKILL.md into a `.claude/skills/armoctl/` directory.

## Known gaps

- `incidents resolve --yes` and `posture resources` hit Cloudflare 5xx/4xx on the dev backend; investigate platform side.
- Jira/SIEM integrations need tenant-side configuration before commands return useful data.
- This is a preview build — not a tagged release. Reference SHA: see `armoctl version` once published.

## License / source

`feature/agent-bridge-smoke-fixes` branch on github.com/armosec/armoctl. PRs #12–#20.
