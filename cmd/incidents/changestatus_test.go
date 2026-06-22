package incidents

import "testing"

func TestNormalizeStatus(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"Dismissed", "Dismissed", false},
		{"dismissed", "Dismissed", false},
		{"  resolved ", "Resolved", false},
		{"INVESTIGATING", "Investigating", false},
		{"open", "Open", false},
		{"", "", true},
		{"bogus", "", true},
	}
	for _, c := range cases {
		got, err := normalizeStatus(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("normalizeStatus(%q): expected error, got %q", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("normalizeStatus(%q): unexpected error %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("normalizeStatus(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
