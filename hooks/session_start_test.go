// hooks/session_start_test.go
package hooks_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// hostEnvs are the variables each agent host provides when it runs the
// command from hooks/hooks.json. Claude Code and Gemini CLI substitute their
// placeholder in the command string; Copilot CLI exports all three root
// variables into the hook's environment. Setting them as env vars and letting
// bash expand the command reproduces both behaviours.
var hostEnvs = map[string][]string{
	"claude":  {"CLAUDE_PLUGIN_ROOT"},
	"gemini":  {"extensionPath"},
	"copilot": {"CLAUDE_PLUGIN_ROOT", "PLUGIN_ROOT", "COPILOT_PLUGIN_ROOT"},
}

// fixture is a throwaway plugin install: a copy of the real hook script and
// hooks.json under a fake plugin root, a stub bin dir, and an isolated HOME so
// installs never touch the developer's ~/.local/bin.
type fixture struct {
	root    string // plugin root
	stubDir string // holds stub armoctl / curl, prepended to PATH
	home    string
}

func newFixture(t *testing.T, pluginVersion string) *fixture {
	t.Helper()
	f := &fixture{root: t.TempDir(), stubDir: t.TempDir(), home: t.TempDir()}
	repoRoot, _ := filepath.Abs("..")
	for _, rel := range []string{"hooks/session-start.sh", "hooks/hooks.json"} {
		body, err := os.ReadFile(filepath.Join(repoRoot, rel))
		if err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(f.root, rel), string(body), 0o755)
	}
	writeFile(t, filepath.Join(f.root, ".claude-plugin", "plugin.json"),
		`{"name":"armoctl","version":"`+pluginVersion+`"}`, 0o644)
	return f
}

func writeFile(t *testing.T, path, body string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), mode); err != nil {
		t.Fatal(err)
	}
}

// stubArmoctl writes a fake armoctl. versionOut is what `armoctl version`
// prints in its "Version:" line (on stderr, like the real binary); flagOut is
// the full `armoctl --version` line.
// `armoctl update` records the call in update_called.
func (f *fixture) stubArmoctl(t *testing.T, versionOut, flagOut string) {
	t.Helper()
	script := `#!/usr/bin/env bash
case "$1" in
  version) printf 'armoctl\n\n  Version:     %s\n' >&2 '` + versionOut + `' ;;
  --version) echo '` + flagOut + `' ;;
  update) echo updated > '` + filepath.Join(f.stubDir, "update_called") + `' ;;
esac
exit 0
`
	writeFile(t, filepath.Join(f.stubDir, "armoctl"), script, 0o755)
}

// stubCurl writes a fake curl that serves an install.sh. When ok, the served
// script drops a stub armoctl into the --dir it is given and, like the real
// installer, prints progress on stdout; otherwise curl fails.
func (f *fixture) stubCurl(t *testing.T, ok bool) {
	t.Helper()
	script := "#!/usr/bin/env bash\nexit 22\n"
	if ok {
		script = `#!/usr/bin/env bash
cat <<'EOF'
echo "Downloading armoctl..."
while [ $# -gt 0 ]; do case "$1" in --dir) dir=$2; shift ;; esac; shift; done
mkdir -p "$dir" && printf '#!/usr/bin/env bash\nexit 0\n' > "$dir/armoctl" && chmod +x "$dir/armoctl"
EOF
`
	}
	writeFile(t, filepath.Join(f.stubDir, "curl"), script, 0o755)
}

func (f *fixture) updateCalled() bool {
	_, err := os.Stat(filepath.Join(f.stubDir, "update_called"))
	return err == nil
}

// baseEnv is a minimal environment: the stub dir first on PATH, then the
// system dirs for bash/jq/awk — but never the developer's own armoctl.
func (f *fixture) baseEnv(extraPath ...string) []string {
	path := strings.Join(append([]string{f.stubDir}, extraPath...), ":") + ":/usr/local/bin:/usr/bin:/bin"
	if jq, err := exec.LookPath("jq"); err == nil {
		path += ":" + filepath.Dir(jq)
	}
	return []string{"HOME=" + f.home, "PATH=" + path}
}

// run executes the hook script directly with extra env vars. It returns
// stdout and stderr separately: stdout is what hosts parse as hook JSON.
func (f *fixture) run(t *testing.T, env []string, extraPath ...string) (stdout, stderr string) {
	t.Helper()
	return f.exec(t, exec.Command("bash", filepath.Join(f.root, "hooks", "session-start.sh")), env, extraPath...)
}

func (f *fixture) exec(t *testing.T, cmd *exec.Cmd, env []string, extraPath ...string) (string, string) {
	t.Helper()
	var out, errb strings.Builder
	cmd.Stdout, cmd.Stderr = &out, &errb
	cmd.Env = append(f.baseEnv(extraPath...), env...)
	if err := cmd.Run(); err != nil {
		t.Fatalf("hook must always exit 0, got %v\nstdout: %s\nstderr: %s", err, out.String(), errb.String())
	}
	return out.String(), errb.String()
}

// hookCommand returns the SessionStart command string from hooks.json.
func (f *fixture) hookCommand(t *testing.T) string {
	t.Helper()
	var cfg struct {
		Hooks map[string][]struct {
			Hooks []struct {
				Command string `json:"command"`
			} `json:"hooks"`
		} `json:"hooks"`
	}
	body, err := os.ReadFile(filepath.Join(f.root, "hooks", "hooks.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(body, &cfg); err != nil {
		t.Fatalf("hooks.json is not valid JSON: %v", err)
	}
	entries := cfg.Hooks["SessionStart"]
	if len(entries) != 1 || len(entries[0].Hooks) != 1 {
		t.Fatalf("expected exactly one SessionStart command, got %+v", entries)
	}
	return entries[0].Hooks[0].Command
}

// assertContextJSON checks stdout carries the message in both shapes: the
// top-level key Copilot CLI reads and the hookSpecificOutput key Claude Code
// and Gemini CLI read.
func assertContextJSON(t *testing.T, stdout, want string) {
	t.Helper()
	var msg struct {
		AdditionalContext  string `json:"additionalContext"`
		HookSpecificOutput struct {
			HookEventName     string `json:"hookEventName"`
			AdditionalContext string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal([]byte(stdout), &msg); err != nil {
		t.Fatalf("stdout is not a single JSON object: %v\n%s", err, stdout)
	}
	if !strings.Contains(msg.AdditionalContext, want) {
		t.Errorf("top-level additionalContext = %q, want it to contain %q", msg.AdditionalContext, want)
	}
	if msg.HookSpecificOutput.HookEventName != "SessionStart" || msg.HookSpecificOutput.AdditionalContext != msg.AdditionalContext {
		t.Errorf("hookSpecificOutput does not mirror additionalContext: %+v", msg.HookSpecificOutput)
	}
}

// The hooks.json command must resolve to the real script under every host's
// variable set. A version mismatch makes the hook call `armoctl update`, which
// proves the script ran and read the fixture's plugin.json.
func TestHookCommand_ResolvesUnderEachHost(t *testing.T) {
	for host, vars := range hostEnvs {
		t.Run(host, func(t *testing.T) {
			f := newFixture(t, "0.0.8")
			f.stubArmoctl(t, "0.0.7", "armoctl version 0.0.7")
			var env []string
			for _, v := range vars {
				env = append(env, v+"="+f.root)
			}
			f.exec(t, exec.Command("bash", "-c", f.hookCommand(t)), env)
			if !f.updateCalled() {
				t.Errorf("hook did not run from %s under %s variables", f.root, host)
			}
		})
	}
}

// With no host variable the command collapses to /hooks/session-start.sh.
// That is the documented failure mode, not something hosts do; the script
// itself still works when invoked by path (see TestHook_IgnoresInjectedRoot).
func TestHookCommand_NoHostVars_CollapsesToRootPath(t *testing.T) {
	f := newFixture(t, "0.0.7")
	cmd := exec.Command("bash", "-c", "echo "+f.hookCommand(t))
	cmd.Env = f.baseEnv()
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(out)); got != "/hooks/session-start.sh" {
		t.Errorf("expanded command = %q", got)
	}
}

// The script resolves the plugin root from its own path, so a stale or wrong
// injected variable can't point it at another plugin.json.
func TestHook_IgnoresInjectedRoot(t *testing.T) {
	f := newFixture(t, "0.0.7")
	f.stubArmoctl(t, "0.0.7", "armoctl version 0.0.7")
	wrong := newFixture(t, "9.9.9").root
	stdout, _ := f.run(t, []string{"CLAUDE_PLUGIN_ROOT=" + wrong, "extensionPath=" + wrong, "PLUGIN_ROOT=" + wrong})
	if f.updateCalled() {
		t.Errorf("hook read the injected root's plugin.json instead of its own")
	}
	if stdout != "" {
		t.Errorf("expected no stdout, got %q", stdout)
	}
}

func TestHook_VersionMatch_NoOpAndQuiet(t *testing.T) {
	f := newFixture(t, "0.0.7")
	f.stubArmoctl(t, "0.0.7", "armoctl version 0.0.7")
	stdout, stderr := f.run(t, nil)
	if f.updateCalled() {
		t.Errorf("update should NOT have been called when versions match")
	}
	if stdout != "" || stderr != "" {
		t.Errorf("expected a silent no-op, got stdout=%q stderr=%q", stdout, stderr)
	}
}

func TestHook_VersionMismatch_RunsUpdate(t *testing.T) {
	f := newFixture(t, "0.0.8")
	f.stubArmoctl(t, "0.0.7", "armoctl version 0.0.7")
	stdout, stderr := f.run(t, nil)
	if !f.updateCalled() {
		t.Errorf("update should have been called when versions differ. stderr: %s", stderr)
	}
	if !strings.Contains(stderr, "differs from plugin") {
		t.Errorf("expected mismatch message, got: %s", stderr)
	}
	if stdout != "" {
		t.Errorf("update output must not reach stdout (hosts parse it as JSON), got %q", stdout)
	}
}

// Released binaries up to v0.0.13 print "unknown (built from source)" for
// --version. The hook must use `armoctl version` so those binaries don't look
// out of date on every session.
func TestHook_LegacyVersionFlag_NoSpuriousUpdate(t *testing.T) {
	f := newFixture(t, "0.0.13")
	f.stubArmoctl(t, "0.0.13", "armoctl version unknown (built from source)")
	f.run(t, nil)
	if f.updateCalled() {
		t.Errorf("update called although `armoctl version` matches the plugin")
	}
}

// A locally built binary reports "dev" (go build) or a git-describe version
// (make); updating would overwrite it with a release.
func TestHook_DevBuild_SkipsUpdate(t *testing.T) {
	for _, v := range []string{"dev", "v0.0.13-dirty", "v0.0.12-4-gabc1234"} {
		t.Run(v, func(t *testing.T) {
			f := newFixture(t, "0.0.13")
			f.stubArmoctl(t, v, "armoctl version "+v)
			stdout, stderr := f.run(t, nil)
			if f.updateCalled() {
				t.Errorf("update must not replace a local build")
			}
			if stdout != "" || !strings.Contains(stderr, "not a release build") {
				t.Errorf("stdout=%q stderr=%q", stdout, stderr)
			}
		})
	}
}

func TestHook_BinaryMissing_Installs(t *testing.T) {
	f := newFixture(t, "0.0.7")
	f.stubCurl(t, true)
	installDir := filepath.Join(f.home, ".local", "bin")
	stdout, stderr := f.run(t, nil, installDir)
	if _, err := os.Stat(filepath.Join(installDir, "armoctl")); err != nil {
		t.Fatalf("armoctl was not installed into %s. stderr: %s", installDir, stderr)
	}
	if !strings.Contains(stderr, "installing v0.0.7") {
		t.Errorf("expected install message, got: %s", stderr)
	}
	assertContextJSON(t, stdout, "just installed")
}

func TestHook_BinaryMissing_InstallDirNotOnPath(t *testing.T) {
	f := newFixture(t, "0.0.7")
	f.stubCurl(t, true)
	stdout, _ := f.run(t, nil)
	assertContextJSON(t, stdout, "is not on $PATH")
}

func TestHook_BinaryMissing_InstallFails(t *testing.T) {
	f := newFixture(t, "0.0.7")
	f.stubCurl(t, false)
	stdout, stderr := f.run(t, nil)
	if !strings.Contains(stderr, "armoctl install failed") {
		t.Errorf("expected failure message, got: %s", stderr)
	}
	assertContextJSON(t, stdout, "auto-install failed")
}

func TestHook_MissingPluginJSON_ExitsCleanly(t *testing.T) {
	f := newFixture(t, "0.0.7")
	if err := os.Remove(filepath.Join(f.root, ".claude-plugin", "plugin.json")); err != nil {
		t.Fatal(err)
	}
	_, stderr := f.run(t, nil)
	if !strings.Contains(stderr, "could not read plugin.json") {
		t.Errorf("stderr: %s", stderr)
	}
}
