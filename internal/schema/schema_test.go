package schema

import (
	"strings"
	"testing"
)

func TestListEnumeratesEmbeddedResources(t *testing.T) {
	names := List()
	found := false
	for _, n := range names {
		if n == "incidents" {
			found = true
		}
	}
	if !found {
		t.Fatalf("incidents not in List(): %v", names)
	}
}

func TestGetReturnsJSONSchemaBytes(t *testing.T) {
	b, err := Get("incidents")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"$schema"`) && !strings.Contains(string(b), `"type"`) {
		t.Fatalf("schema content missing JSON schema markers: %s", string(b))
	}
}
