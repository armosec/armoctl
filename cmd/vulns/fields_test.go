package vulns

import (
	"bytes"
	"strings"
	"testing"
)

func TestCheatsheetCoversAllScopes(t *testing.T) {
	cs := Cheatsheet()
	for _, want := range []string{"workloads", "images", "components", "cves", "hosts"} {
		if len(cs[want]) < 4 {
			t.Errorf("scope %q: cheatsheet too small (%d)", want, len(cs[want]))
		}
	}
}

func TestFieldsCmd_AllScopes(t *testing.T) {
	cmd := FieldsCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs(nil)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"workloads", "images", "components", "cves", "hosts"} {
		if !strings.Contains(buf.String(), "### "+want) {
			t.Errorf("output missing scope header for %q", want)
		}
	}
}

func TestFieldsCmd_OneScope(t *testing.T) {
	cmd := FieldsCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"workloads"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "wlid") {
		t.Fatalf("output missing wlid: %s", buf.String())
	}
}

func TestFieldsCmd_ExtraArgsRejected(t *testing.T) {
	cmd := FieldsCmd()
	cmd.SetArgs([]string{"workloads", "extra"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error for extra positional arg, got nil")
	}
}
