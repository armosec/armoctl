package incidents

import (
	"bytes"
	"strings"
	"testing"
)

func TestFieldsCheatsheetIsNonEmpty(t *testing.T) {
	cheatsheet := Cheatsheet()
	if len(cheatsheet) < 5 {
		t.Fatalf("cheatsheet too small: %d", len(cheatsheet))
	}
	have := map[string]bool{}
	for _, f := range cheatsheet {
		have[f.Name] = true
	}
	for _, want := range []string{"guid", "name", "severity"} {
		if !have[want] {
			t.Errorf("cheatsheet missing %q", want)
		}
	}
}

func TestFieldsCommandPrintsCheatsheet(t *testing.T) {
	cmd := FieldsCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs(nil)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "guid") {
		t.Fatalf("output missing guid: %q", buf.String())
	}
}
