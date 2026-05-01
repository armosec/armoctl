package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

type item struct {
	Guid string `json:"guid"`
	Name string `json:"name"`
}

func TestRenderJSONList(t *testing.T) {
	r := List{
		Items:    []any{item{"a", "alpha"}, item{"b", "beta"}},
		Total:    2,
		Page:     1,
		PageSize: 50,
	}
	var buf bytes.Buffer
	if err := Render(&buf, r, Options{Format: "json"}); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("not JSON: %v: %q", err, buf.String())
	}
	if got["total"].(float64) != 2 {
		t.Fatalf("total = %v", got["total"])
	}
	items := got["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("items len = %d", len(items))
	}
}

func TestRenderJSONGet(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, Get{Object: item{"x", "ex"}}, Options{Format: "json"}); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("not JSON: %v: %q", err, buf.String())
	}
	if got["guid"] != "x" {
		t.Fatalf("guid = %v", got["guid"])
	}
}

func TestRenderJSONMutation(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, Mutation{Result: "ok", Changed: true}, Options{Format: "json"}); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("not JSON: %v: %q", err, buf.String())
	}
	if got["changed"] != true {
		t.Fatalf("changed = %v", got["changed"])
	}
}

func TestRenderYAMLList(t *testing.T) {
	r := List{Items: []any{item{"a", "alpha"}}, Total: 1}
	var buf bytes.Buffer
	if err := Render(&buf, r, Options{Format: "yaml"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "items:") || !strings.Contains(buf.String(), "alpha") {
		t.Fatalf("yaml unexpected:\n%s", buf.String())
	}
}

func TestRenderNDJSONList(t *testing.T) {
	r := List{Items: []any{item{"a", "alpha"}, item{"b", "beta"}}, Total: 2}
	var buf, errBuf bytes.Buffer
	if err := Render(&buf, r, Options{Format: "ndjson", Stderr: &errBuf}); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("ndjson lines = %d: %q", len(lines), buf.String())
	}
	if !strings.Contains(errBuf.String(), `"total":2`) {
		t.Fatalf("ndjson stderr meta missing total: %q", errBuf.String())
	}
}

func TestRenderCSVList(t *testing.T) {
	r := List{Items: []any{item{"a", "alpha"}, item{"b", "beta"}}, Total: 2}
	var buf bytes.Buffer
	if err := Render(&buf, r, Options{Format: "csv"}); err != nil {
		t.Fatal(err)
	}
	csv := buf.String()
	if !strings.HasPrefix(csv, "guid,name\n") {
		t.Fatalf("csv header bad: %q", csv)
	}
	if !strings.Contains(csv, "a,alpha\n") {
		t.Fatalf("csv row missing: %q", csv)
	}
}

func TestRenderTableList(t *testing.T) {
	r := List{Items: []any{item{"a", "alpha"}}, Total: 1}
	var buf bytes.Buffer
	if err := Render(&buf, r, Options{Format: "table"}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "alpha") {
		t.Fatalf("table missing alpha:\n%s", out)
	}
}
