package safety

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAuditAppend(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ARMOCTL_AUDIT_LOG", filepath.Join(dir, "audit.log"))

	if err := AuditAppend(Entry{
		Command:   "incidents.resolve",
		URL:       "POST /runtime/incidents/x/resolve",
		Status:    200,
		RequestID: "r1",
	}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "audit.log"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "incidents.resolve") || !strings.Contains(string(b), "requestId=r1") {
		t.Fatalf("audit line bad: %q", string(b))
	}
}
