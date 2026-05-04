#!/usr/bin/env bash
# hooks/session-start.sh
#
# SessionStart hook for the armoctl plugin/extension. Invoked by both
# Claude Code (via ${CLAUDE_PLUGIN_ROOT} substitution) and Gemini CLI
# (via ${extensionPath} substitution) — both auto-discover hooks/hooks.json
# at the extension root and use the same JSON schema.
#
# - Ensures the armoctl binary is installed.
# - If installed, ensures it matches the plugin's pinned version.
# - Never blocks session start: prints actionable output and exits 0
#   on failure paths.
#
# When something goes wrong the hook also emits a JSON message on stdout
# via the SessionStart hookSpecificOutput contract so the user actually
# sees the failure (stderr alone tends to be silent in Claude; Gemini may
# ignore the JSON shape but the stderr path still surfaces).

# Resolve the extension root from whichever variable the host set, falling
# back to a path derived from $0 for standalone invocation.
PLUGIN_ROOT="${CLAUDE_PLUGIN_ROOT:-${extensionPath:-$(cd "$(dirname "$0")/.." && pwd)}}"

# emit_context prints a SessionStart hookSpecificOutput JSON blob on stdout.
# Claude Code surfaces additionalContext as a system note in the session.
emit_context() {
    local msg=$1
    # JSON-escape: backslashes, double quotes, newlines.
    msg=${msg//\\/\\\\}
    msg=${msg//\"/\\\"}
    msg=${msg//$'\n'/\\n}
    printf '{"hookSpecificOutput":{"hookEventName":"SessionStart","additionalContext":"%s"}}\n' "$msg"
}

PLUGIN_VERSION="$(jq -r .version "$PLUGIN_ROOT/.claude-plugin/plugin.json" 2>/dev/null)" || {
    echo "armoctl plugin: could not read plugin.json — skipping version check." >&2
    exit 0
}

INSTALLED=""
if command -v armoctl >/dev/null 2>&1; then
    INSTALLED="$(armoctl --version 2>/dev/null | awk '{print $NF}')"
fi

# Strip leading 'v' if present for comparison.
norm() { echo "${1#v}"; }

INSTALL_URL="https://package-distribution.armosec.io/armoctl/install.sh"

# Install location for hook-driven installs: a user-writable dir so we
# never need sudo (which fails non-interactively under Claude Code).
INSTALL_DIR="$HOME/.local/bin"

if [ -z "$INSTALLED" ]; then
    echo "armoctl not found — installing v${PLUGIN_VERSION} into ${INSTALL_DIR}…" >&2
    mkdir -p "$INSTALL_DIR"
    # Pin to the plugin's declared version so the binary matches what
    # this plugin tag was tested against. Without --version, install.sh
    # would fetch 'latest', which can drift ahead of the plugin tag.
    if ! curl -fsSL "$INSTALL_URL" | bash -s -- --version "v${PLUGIN_VERSION#v}" --dir "$INSTALL_DIR"; then
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
elif [ "$(norm "$INSTALLED")" != "$(norm "$PLUGIN_VERSION")" ]; then
    echo "armoctl ${INSTALLED} differs from plugin ${PLUGIN_VERSION} — running 'armoctl update'…" >&2
    if ! armoctl update; then
        echo "armoctl update failed; continuing with ${INSTALLED}." >&2
        emit_context "armoctl is at ${INSTALLED} but the plugin expects ${PLUGIN_VERSION}. Auto-update failed; consider running 'armoctl update' manually."
    fi
fi

exit 0
