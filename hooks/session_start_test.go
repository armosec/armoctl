// hooks/session_start_test.go
package hooks_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runHook invokes session-start.sh with PATH set so the stub binaries are found.
// Returns combined stdout/stderr.
func runHook(t *testing.T, stubPath, pluginRoot string) (string, error) {
	t.Helper()
	repoRoot, _ := filepath.Abs("..")
	cmd := exec.Command("bash", filepath.Join(repoRoot, "hooks/session-start.sh"))
	cmd.Env = append(os.Environ(),
		"PATH="+stubPath+":"+os.Getenv("PATH"),
		"CLAUDE_PLUGIN_ROOT="+pluginRoot,
	)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// makeStubArmoctl writes a fake armoctl that prints `armoctl <version>`
// when called with --version. If version is empty, the stub is omitted
// (simulating "binary not on PATH").
func makeStubArmoctl(t *testing.T, dir, version string) {
	t.Helper()
	if version == "" {
		return
	}
	script := "#!/usr/bin/env bash\nif [ \"$1\" = \"--version\" ]; then echo armoctl " + version + "; exit 0; fi\nif [ \"$1\" = \"update\" ]; then echo updated > " + filepath.Join(dir, "update_called") + "; exit 0; fi\nexit 0\n"
	path := filepath.Join(dir, "armoctl")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
}

func makePluginJSON(t *testing.T, dir, version string) string {
	t.Helper()
	pluginDir := filepath.Join(dir, ".claude-plugin")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"name":"armoctl","version":"` + version + `"}`
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestHook_VersionMatch_NoOp(t *testing.T) {
	stubDir := t.TempDir()
	pluginRoot := makePluginJSON(t, t.TempDir(), "0.0.7")
	makeStubArmoctl(t, stubDir, "0.0.7")

	out, err := runHook(t, stubDir, pluginRoot)
	if err != nil {
		t.Fatalf("hook failed: %v\n%s", err, out)
	}
	if _, statErr := os.Stat(filepath.Join(stubDir, "update_called")); statErr == nil {
		t.Errorf("update should NOT have been called when versions match")
	}
}

func TestHook_VersionMismatch_RunsUpdate(t *testing.T) {
	stubDir := t.TempDir()
	pluginRoot := makePluginJSON(t, t.TempDir(), "0.0.8")
	makeStubArmoctl(t, stubDir, "0.0.7")

	out, err := runHook(t, stubDir, pluginRoot)
	if err != nil {
		t.Fatalf("hook failed: %v\n%s", err, out)
	}
	if _, statErr := os.Stat(filepath.Join(stubDir, "update_called")); statErr != nil {
		t.Errorf("update should have been called when versions differ. Output: %s", out)
	}
	if !strings.Contains(out, "differs from plugin") {
		t.Errorf("expected mismatch message, got: %s", out)
	}
}

func TestHook_BinaryMissing_PrintsInstallHint(t *testing.T) {
	stubDir := t.TempDir() // no armoctl stub
	pluginRoot := makePluginJSON(t, t.TempDir(), "0.0.7")

	out, err := runHook(t, stubDir, pluginRoot)
	// Hook may exit 0 (graceful) even when curl install fails. We only
	// assert it doesn't blow up the session and prints something useful.
	if err != nil && !strings.Contains(out, "armoctl install failed") && !strings.Contains(out, "installing v0.0.7") {
		t.Fatalf("hook failed unexpectedly: %v\n%s", err, out)
	}
	if !strings.Contains(out, "armoctl") {
		t.Errorf("expected armoctl-related output, got: %s", out)
	}
}
