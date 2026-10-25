package main

import (
	"encoding/json"
	"os"
	"regexp"
	"testing"
)

// TestPluginManifests_VersionsMatch guards the plugin manifests every agent
// host reads (Claude Code and Copilot CLI read .claude-plugin/, Gemini CLI
// reads gemini-extension.json). They must be valid JSON and carry the same
// version, which the SessionStart hook pins the binary to. The release
// workflow bumps them together; this catches hand edits that drift.
func TestPluginManifests_VersionsMatch(t *testing.T) {
	semver := regexp.MustCompile(`^\d+\.\d+\.\d+$`)

	readJSON := func(path string, v any) {
		t.Helper()
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(body, v); err != nil {
			t.Fatalf("%s is not valid JSON: %v", path, err)
		}
	}

	var plugin, gemini struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	var marketplace struct {
		Name    string `json:"name"`
		Plugins []struct {
			Name    string `json:"name"`
			Version string `json:"version"`
			Source  string `json:"source"`
		} `json:"plugins"`
	}
	readJSON(".claude-plugin/plugin.json", &plugin)
	readJSON("gemini-extension.json", &gemini)
	readJSON(".claude-plugin/marketplace.json", &marketplace)

	if !semver.MatchString(plugin.Version) {
		t.Fatalf(".claude-plugin/plugin.json version %q is not MAJOR.MINOR.PATCH", plugin.Version)
	}
	versions := map[string]string{"gemini-extension.json": gemini.Version}
	for _, p := range marketplace.Plugins {
		if p.Name == plugin.Name {
			versions[".claude-plugin/marketplace.json plugins["+p.Name+"]"] = p.Version
		}
	}
	if len(versions) != 2 {
		t.Fatalf("marketplace.json has no plugin entry named %q", plugin.Name)
	}
	for file, v := range versions {
		if v != plugin.Version {
			t.Errorf("%s version %q != .claude-plugin/plugin.json version %q", file, v, plugin.Version)
		}
	}
}
