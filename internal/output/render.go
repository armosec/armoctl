package output

import (
	"encoding/json"
	"fmt"
	"io"
)

// Render writes r in the requested format.
func Render(w io.Writer, r Result, o Options) error {
	switch o.Format {
	case "", "json":
		return renderJSON(w, r)
	default:
		return fmt.Errorf("unsupported output format %q", o.Format)
	}
}

func renderJSON(w io.Writer, r Result) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	switch v := r.(type) {
	case Get:
		return enc.Encode(v.Object)
	default:
		return enc.Encode(r)
	}
}
