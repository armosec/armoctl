package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestProjectKeepsOnlySelectedFields(t *testing.T) {
	in := map[string]any{"guid": "a", "name": "alpha", "deep": map[string]any{"k": "v"}}
	got := project(in, []string{"guid", "deep.k"})
	m := got.(map[string]any)
	if m["guid"] != "a" {
		t.Fatalf("guid lost: %v", m)
	}
	if d, ok := m["deep"].(map[string]any); !ok || d["k"] != "v" {
		t.Fatalf("deep.k lost: %v", m)
	}
	if _, has := m["name"]; has {
		t.Fatalf("name should be projected away: %v", m)
	}
}

func TestRenderListAppliesSummary(t *testing.T) {
	r := List{
		Items: []any{
			map[string]any{"guid": "a", "name": "alpha", "noise": map[string]any{"big": "data"}},
		},
		Total: 1,
	}
	var buf bytes.Buffer
	o := Options{Format: "json", SummaryFields: []string{"guid", "name"}}
	if err := Render(&buf, r, o); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "noise") {
		t.Fatalf("summary did not strip noise: %s", buf.String())
	}
}

func TestRenderListFullDisablesSummary(t *testing.T) {
	r := List{
		Items: []any{
			map[string]any{"guid": "a", "noise": "kept"},
		},
	}
	var buf bytes.Buffer
	o := Options{Format: "json", SummaryFields: []string{"guid"}, Full: true}
	if err := Render(&buf, r, o); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "noise") {
		t.Fatalf("--full should keep noise: %s", buf.String())
	}
}

func TestRenderListFieldsOverridesSummary(t *testing.T) {
	r := List{Items: []any{map[string]any{"guid": "a", "name": "alpha", "extra": "x"}}}
	var buf bytes.Buffer
	o := Options{Format: "json", SummaryFields: []string{"guid"}, Fields: []string{"guid", "extra"}}
	if err := Render(&buf, r, o); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"extra"`) {
		t.Fatalf("--fields should keep extra: %s", buf.String())
	}
	if strings.Contains(buf.String(), `"name"`) {
		t.Fatalf("--fields should drop name: %s", buf.String())
	}
}
