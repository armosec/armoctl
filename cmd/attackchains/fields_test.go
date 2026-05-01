package attackchains

import (
	"bytes"
	"strings"
	"testing"
)

func TestCheatsheetNotEmpty(t *testing.T) {
	cs := Cheatsheet()
	if len(cs) < 3 {
		t.Errorf("cheatsheet too small (%d)", len(cs))
	}
}

func TestFieldsCmd(t *testing.T) {
	cmd := FieldsCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs(nil)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "guid") {
		t.Errorf("output missing guid: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "Field cheatsheet:") {
		t.Errorf("output missing header: %s", buf.String())
	}
}
