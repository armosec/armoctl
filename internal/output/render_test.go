package output

import (
	"bytes"
	"encoding/json"
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
