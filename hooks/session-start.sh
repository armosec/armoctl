#!/usr/bin/env bash
# hooks/session-start.sh
#
# SessionStart hook for the armoctl Claude plugin.
# - Ensures the armoctl binary is installed.
# - If installed, ensures it matches the plugin's pinned version.
# - Never blocks session start: prints actionable output and exits 0
#   on failure paths.

set -e

PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"
PLUGIN_VERSION="$(jq -r .version "$PLUGIN_ROOT/.claude-plugin/plugin.json")"

INSTALLED=""
if command -v armoctl >/dev/null 2>&1; then
    INSTALLED="$(armoctl --version 2>/dev/null | awk '{print $NF}')"
fi

# Strip leading 'v' if present for comparison.
norm() { echo "${1#v}"; }

INSTALL_URL="https://package-distribution.armosec.io/armoctl/install.sh"
if [ -z "$INSTALLED" ]; then
    echo "armoctl not found — installing v${PLUGIN_VERSION}…" >&2
    if ! curl -fsSL "$INSTALL_URL" | bash; then
        echo "armoctl install failed; the armoctl skill will not work this session." >&2
        echo "Install manually: curl -fsSL $INSTALL_URL | bash" >&2
        exit 0
    fi
elif [ "$(norm "$INSTALLED")" != "$(norm "$PLUGIN_VERSION")" ]; then
    echo "armoctl ${INSTALLED} differs from plugin ${PLUGIN_VERSION} — running 'armoctl update'…" >&2
    armoctl update || echo "armoctl update failed; continuing with ${INSTALLED}." >&2
fi

exit 0
