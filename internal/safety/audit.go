// Package safety wraps mutating CLI commands with dry-run, confirmation,
// and audit logging.
package safety

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Entry is a single audit log record.
type Entry struct {
	Command   string
	URL       string
	Status    int
	RequestID string
	Args      string // already-redacted single-line args
}

// AuditAppend writes one line to the audit log.
// Path: $ARMOCTL_AUDIT_LOG, else ~/.armoctl/audit.log.
func AuditAppend(e Entry) error {
	path := os.Getenv("ARMOCTL_AUDIT_LOG")
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		path = filepath.Join(home, ".armoctl", "audit.log")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	line := fmt.Sprintf("%s %s args=%q url=%s status=%d requestId=%s\n",
		time.Now().UTC().Format(time.RFC3339), e.Command, e.Args, e.URL, e.Status, e.RequestID)
	_, err = f.WriteString(line)
	return err
}
