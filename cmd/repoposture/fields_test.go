package repoposture

import (
	"bytes"
	"strings"
	"testing"
)

func TestCheatsheetCoversAllScopes(t *testing.T) {
	cs := Cheatsheet()
	for _, want := range []string{"repositories", "files", "resources", "failed-controls"} {
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
	for _, want := range []string{"repositories", "files", "resources", "failed-controls"} {
		if !strings.Contains(buf.String(), "### "+want) {
			t.Errorf("output missing scope header for %q", want)
		}
	}
}

func TestFieldsCmd_OneScope(t *testing.T) {
	cmd := FieldsCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"repositories"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "owner") {
		t.Fatalf("output missing owner: %s", buf.String())
	}
}
