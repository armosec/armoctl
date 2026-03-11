package version

import (
	"bytes"
	"os"
	"testing"
)

func TestPrintUpdateBanner_NilInfo(t *testing.T) {
	// Should not panic with nil info
	PrintUpdateBanner(nil)
}

func TestPrintUpdateBanner_NoUpdate(t *testing.T) {
	info := &UpdateInfo{
		ArmoCtlCurrent: "v1.0.0",
		ArmoCtlLatest:  "v1.0.0",
		HasUpdate:      false,
	}

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	PrintUpdateBanner(info)

	_ = w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	// Should not print anything when no update
	if buf.Len() > 0 {
		t.Errorf("PrintUpdateBanner() printed output when no update: %s", buf.String())
	}
}

func TestPadding_VariousLengths(t *testing.T) {
	tests := []struct {
		current string
		latest  string
	}{
		{"v0.0.1", "v0.0.2"},
		{"v0.0.1", "v10.10.10"},
		{"v1.0.0", "v1.0.0"},
		{"dev", "v1.0.0"},
		{"", "v1.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.current+"_"+tt.latest, func(t *testing.T) {
			result := padding(tt.current, tt.latest)
			// Should always return at least one space
			if len(result) < 1 {
				t.Errorf("padding(%q, %q) returned %d chars, want >= 1", tt.current, tt.latest, len(result))
			}
		})
	}
}
