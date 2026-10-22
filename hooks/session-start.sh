#!/usr/bin/env bash
# hooks/session-start.sh
#
# SessionStart hook for the armoctl plugin/extension. One hooks/hooks.json
# serves every host; each expands the command path its own way:
#   - Claude Code:      ${CLAUDE_PLUGIN_ROOT}
#   - Gemini CLI:       ${extensionPath}
#   - GitHub Copilot CLI: exports CLAUDE_PLUGIN_ROOT (plus PLUGIN_ROOT and
#     COPILOT_PLUGIN_ROOT) into the hook's environment, so the Claude form
#     resolves there too.
#
# - Ensures the armoctl binary is installed.
# - If installed, ensures it matches the plugin's pinned version.
# - Quiet on success. Never blocks session start: prints actionable output
#   and exits 0 on failure paths.
#
# When something goes wrong the hook also emits a JSON message on stdout so
# the user actually sees the failure (stderr alone tends to be silent).

# Resolve the plugin root from this script's own location rather than from
# host-injected variables: hooks/session-start.sh always sits one level below
# the root, whichever host invoked it.
PLUGIN_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# emit_context prints a SessionStart context message on stdout. The object
# carries the message twice: Claude Code and Gemini CLI read
# hookSpecificOutput.additionalContext, Copilot CLI reads the top-level
# additionalContext. Each host ignores the key it doesn't know.
emit_context() {
    local msg=$1
    # JSON-escape: backslashes, double quotes, newlines.
    msg=${msg//\\/\\\\}
    msg=${msg//\"/\\\"}
    msg=${msg//$'\n'/\\n}
    printf '{"additionalContext":"%s","hookSpecificOutput":{"hookEventName":"SessionStart","additionalContext":"%s"}}\n' "$msg" "$msg"
}

PLUGIN_VERSION="$(jq -r .version "$PLUGIN_ROOT/.claude-plugin/plugin.json" 2>/dev/null)"
if [ -z "$PLUGIN_VERSION" ] || [ "$PLUGIN_VERSION" = "null" ]; then
    echo "armoctl plugin: could not read plugin.json — skipping version check." >&2
    exit 0
fi

# installed_version prints the installed binary's version. `armoctl version`
# is the reliable source: released binaries up to v0.0.13 print
# "unknown (built from source)" for --version, which would look like drift
# on every session. It writes to stderr, hence 2>&1.
installed_version() {
    local v
    v="$(armoctl version 2>&1 | awk '/Version:/ {print $2; exit}')"
    if [ -z "$v" ]; then
        v="$(armoctl --version 2>/dev/null | awk '{print $NF}')"
    fi
    echo "$v"
}

INSTALLED=""
if command -v armoctl >/dev/null 2>&1; then
    INSTALLED="$(installed_version)"
fi

# Strip leading 'v' if present for comparison.
norm() { echo "${1#v}"; }

INSTALL_URL="https://package-distribution.armosec.io/armoctl/install.sh"

# Install location for hook-driven installs: a user-writable dir so we
# never need sudo (which fails non-interactively under the agent hosts).
INSTALL_DIR="$HOME/.local/bin"

if ! command -v armoctl >/dev/null 2>&1; then
    echo "armoctl not found — installing v${PLUGIN_VERSION} into ${INSTALL_DIR}…" >&2
    mkdir -p "$INSTALL_DIR"
    # Pin to the plugin's declared version so the binary matches what
    # this plugin tag was tested against. Without --version, install.sh
    # would fetch 'latest', which can drift ahead of the plugin tag.
    # pipefail: without it a failed download pipes nothing into bash, which
    # exits 0 and the failure goes unnoticed. The installer's progress output
    # goes to stderr: stdout is reserved for the hook JSON the host parses.
    if ! (set -o pipefail; curl -fsSL "$INSTALL_URL" | bash -s -- --version "v${PLUGIN_VERSION#v}" --dir "$INSTALL_DIR" >&2); then
        echo "armoctl install failed; the armoctl skill will not work this session." >&2
        echo "Install manually: curl -fsSL $INSTALL_URL | bash -s -- --version \"v${PLUGIN_VERSION#v}\" --dir \"$INSTALL_DIR\"" >&2
        emit_context "armoctl auto-install failed. The armoctl skill will not work until the binary is installed. Run: curl -fsSL $INSTALL_URL | bash -s -- --version \"v${PLUGIN_VERSION#v}\" --dir \"\$HOME/.local/bin\""
        exit 0
    fi
    # Warn if the install dir is not on PATH so the user can fix it.
    case ":$PATH:" in
        *":$INSTALL_DIR:"*) ;;
        *)
            echo "Warning: $INSTALL_DIR is not on \$PATH — add it to your shell profile." >&2
            emit_context "armoctl was installed at $INSTALL_DIR/armoctl but that directory is not on \$PATH. Add 'export PATH=\"\$HOME/.local/bin:\$PATH\"' to your shell profile, then run 'armoctl configure' to authenticate."
            exit 0
            ;;
    esac
    emit_context "armoctl v${PLUGIN_VERSION} just installed at ${INSTALL_DIR}/armoctl. If credentials are not yet configured, run 'armoctl configure' to authenticate against the ARMO platform."
elif ! [[ "$(norm "$INSTALLED")" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    # Not a release build: "dev", a local `make` build ("v0.0.13-dirty",
    # "v0.0.13-4-gabc1234") or unparseable. Leave it alone: 'armoctl update'
    # would replace a locally built binary with a release.
    echo "armoctl version '${INSTALLED}' is not a release build — skipping version check." >&2
elif [ "$(norm "$INSTALLED")" != "$(norm "$PLUGIN_VERSION")" ]; then
    echo "armoctl ${INSTALLED} differs from plugin ${PLUGIN_VERSION} — running 'armoctl update'…" >&2
    if ! armoctl update >&2; then
        echo "armoctl update failed; continuing with ${INSTALLED}." >&2
        emit_context "armoctl is at ${INSTALLED} but the plugin expects ${PLUGIN_VERSION}. Auto-update failed; consider running 'armoctl update' manually."
    fi
fi

exit 0
