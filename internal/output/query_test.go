package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderWithQuery_OnList(t *testing.T) {
	r := List{Items: []any{
		map[string]any{"guid": "a", "severity": "high"},
		map[string]any{"guid": "b", "severity": "low"},
	}, Total: 2}
	var buf bytes.Buffer
	o := Options{Format: "json", Query: `.items[] | select(.severity=="high") | .guid`}
	if err := Render(&buf, r, o); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"a"`) || strings.Contains(buf.String(), `"b"`) {
		t.Fatalf("query result unexpected: %q", buf.String())
	}
}
