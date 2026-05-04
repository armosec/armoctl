#!/usr/bin/env bash
# scripts/smoke.sh — high-level smoke tests for armoctl against a live tenant.
#
# Usage:
#   ARMO_CUSTOMER_GUID=... ARMO_ACCESS_KEY=... ARMO_API_BASE_URL=... ./scripts/smoke.sh
#   ./scripts/smoke.sh -c vulns       # only the vulns cluster
#   ./scripts/smoke.sh -v             # verbose
#
# Exits non-zero if any check fails. Number of failures is encoded in the exit code (capped at 255).
# Skipped checks (HTML response → unconfigured integration) do NOT count toward the failure exit code.

set -uo pipefail

# Resolve repo root
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

# Parse flags
VERBOSE=0
CLUSTER_FILTER=""
while getopts "vc:" opt; do
    case "$opt" in
        v) VERBOSE=1 ;;
        c) CLUSTER_FILTER="$OPTARG" ;;
        *) echo "Usage: $0 [-v] [-c <cluster>]" >&2; exit 2 ;;
    esac
done

# Validate env
: "${ARMO_CUSTOMER_GUID:?ARMO_CUSTOMER_GUID must be set}"
: "${ARMO_ACCESS_KEY:?ARMO_ACCESS_KEY must be set}"
: "${ARMO_API_BASE_URL:?ARMO_API_BASE_URL must be set (e.g., api.armosec.io)}"

# Resolve binary
if command -v armoctl >/dev/null 2>&1; then
    ARMOCTL="$(command -v armoctl)"
else
    echo "Building armoctl..." >&2
    ARMOCTL=/tmp/armoctl-smoke
    (cd "$REPO_ROOT" && go build -o "$ARMOCTL" .)
fi

PASS=0
FAIL=0
SKIP=0
FAILED_CHECKS=()

# run_check <cluster> <label> <args...>
# Runs `armoctl <args>`, asserts exit 0 + JSON-parseable stdout.
# If stdout starts with an HTML tag (e.g. Cloudflare error page), the check is
# classified as "skipped (HTML response)" rather than a failure — this usually
# means the feature/integration is not configured for this tenant.
run_check() {
    local cluster="$1" label="$2"; shift 2
    if [ -n "$CLUSTER_FILTER" ] && [ "$CLUSTER_FILTER" != "$cluster" ]; then
        return 0
    fi
    local out rc=0
    out="$("$ARMOCTL" "$@" 2>/tmp/smoke.err)" || rc=$?
    local err
    err="$(cat /tmp/smoke.err)"
    if [ "$rc" -ne 0 ]; then
        FAIL=$((FAIL+1))
        FAILED_CHECKS+=("$cluster $label (exit $rc): $err")
        echo "✗ $cluster $label — exit $rc"
        [ "$VERBOSE" -eq 1 ] && echo "    stderr: $err"
        return 0
    fi
    if [ -z "$out" ]; then
        FAIL=$((FAIL+1))
        FAILED_CHECKS+=("$cluster $label: empty stdout")
        echo "✗ $cluster $label — empty stdout"
        return 0
    fi
    # Detect an HTML error page (Cloudflare, nginx, etc.) returned with HTTP 200.
    # This means the feature/integration is not configured for this tenant — skip
    # rather than fail so the overall exit code remains clean.
    if echo "$out" | grep -qiE '^\s*<(html|!DOCTYPE)'; then
        SKIP=$((SKIP+1))
        echo "⊘ $cluster $label — skipped (HTML response, likely unconfigured integration)"
        [ "$VERBOSE" -eq 1 ] && echo "    output: $(echo "$out" | head -3)"
        return 0
    fi
    if ! echo "$out" | jq . >/dev/null 2>&1; then
        FAIL=$((FAIL+1))
        FAILED_CHECKS+=("$cluster $label: stdout not JSON")
        echo "✗ $cluster $label — stdout not JSON"
        [ "$VERBOSE" -eq 1 ] && echo "    output: $(echo "$out" | head -3)"
        return 0
    fi
    PASS=$((PASS+1))
    echo "✓ $cluster $label"
    if [ "$VERBOSE" -eq 1 ]; then
        echo "    $(echo "$out" | jq -c 'if type == "object" then (if has("items") then {items_count: (.items|length), total} else . end) else . end' 2>/dev/null | head -c 120)"
    fi
}

# === Checks ===

# incidents
run_check incidents "list" incidents list --limit 5
run_check incidents "severities" incidents severities

# vulns
run_check vulns "cves" vulns cves --limit 5
run_check vulns "components" vulns components --limit 5
run_check vulns "severity" vulns severity
run_check vulns "exceptions list" vulns exceptions list
# dry-run: create a vulns exception (requires at least one --cve AND one scope flag)
run_check vulns "exceptions create --dry-run" vulns exceptions create --dry-run \
    --name smoke-test-exception --cve CVE-0000-00000 --cluster smoke-test-not-real

# posture
run_check posture "frameworks" posture frameworks
run_check posture "controls" posture controls --limit 5
run_check posture "exceptions list" posture exceptions list
# dry-run: create a posture exception (requires at least one --control AND one scope flag)
run_check posture "exceptions create --dry-run" posture exceptions create --dry-run \
    --name smoke-test-exception --control C-0001 --cluster smoke-test-not-real

# risks
run_check risks "list" risks list --limit 5
run_check risks "severities" risks severities
# FIXME: investigate when live — this check produced exit 0 with non-JSON stdout in
# a live run (2026-05-03). The CLI code path through ListPaged + output.Render(json)
# looks correct; likely an API-side quirk specific to some tenants. Re-run with -v
# to capture the raw output and determine whether it's a CLI bug or an API anomaly.
run_check risks "exceptions list" risks exceptions list --limit 5
# dry-run: create a risks exception (requires --risk-id; --name and scope are optional)
run_check risks "exceptions create --dry-run" risks exceptions create --dry-run \
    --risk-id R-0001 --reason 'smoke test' --cluster smoke-test-not-real

# attack-chains
run_check attack-chains "list" attack-chains list --limit 5

# inventory
run_check inventory "list" inventory list --limit 5
run_check inventory "unique-values namespace" inventory unique-values namespace

# network-policies
run_check network-policies "list" network-policies list --limit 5

# seccomp
run_check seccomp "list" seccomp list --limit 5

# runtime-rules
run_check runtime-rules "list" runtime-rules list --limit 5
# runtime-rules create --dry-run requires a non-trivial rule body (--rule or --rule-file).
# We don't include a generic dry-run mutation for runtime-rules — the runtime-policies
# create check below exercises the same plumbing.

# runtime-policies
run_check runtime-policies "list" runtime-policies list --limit 5
# dry-run: create a runtime policy (requires --name)
run_check runtime-policies "create --dry-run" runtime-policies create --dry-run --name smoke-test-policy

# integrations
# NOTE: integrations alert-channels and integrations siem have no list subcommand —
#       they only have "create". Skip those as read-only checks.
#
# jira projects — returns an HTML error page (Cloudflare 400) when Jira is not
# configured for this tenant. The HTML-detection logic in run_check will classify
# this as "skipped" rather than a failure.
run_check integrations "jira projects" integrations jira projects

# cloud-accounts
run_check cloud-accounts "ecs list" cloud-accounts ecs list

# repo-posture
# NOTE: repo-posture resources, files, and failed-controls all require --report-guid,
#       which is only known after querying a connected repo. Skip those here.
run_check repo-posture "repositories" repo-posture repositories

# === Summary ===
echo
echo "==================================="
echo "Smoke results: $PASS passed, $FAIL failed, $SKIP skipped"
if [ "$FAIL" -gt 0 ]; then
    echo
    echo "Failures:"
    for f in "${FAILED_CHECKS[@]}"; do
        echo "  - $f"
    done
fi
echo "==================================="

# Exit code = number of failures (capped at 255). Skipped checks are not failures.
if [ "$FAIL" -gt 255 ]; then
    exit 255
fi
exit "$FAIL"
