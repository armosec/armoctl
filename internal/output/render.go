package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

func Render(w io.Writer, r Result, o Options) error {
	switch o.Format {
	case "", "json":
		return renderJSON(w, r)
	case "yaml":
		return renderYAML(w, r)
	case "ndjson":
		return renderNDJSON(w, r, o)
	case "csv":
		return renderCSV(w, r)
	case "table":
		return renderTable(w, r)
	default:
		return fmt.Errorf("unsupported output format %q", o.Format)
	}
}

func renderJSON(w io.Writer, r Result) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if g, ok := r.(Get); ok {
		return enc.Encode(g.Object)
	}
	return enc.Encode(r)
}

func renderYAML(w io.Writer, r Result) error {
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	defer enc.Close()
	if g, ok := r.(Get); ok {
		return enc.Encode(g.Object)
	}
	return enc.Encode(r)
}

func renderNDJSON(w io.Writer, r Result, o Options) error {
	switch v := r.(type) {
	case List:
		enc := json.NewEncoder(w)
		for _, it := range v.Items {
			if err := enc.Encode(it); err != nil {
				return err
			}
		}
		if o.Stderr != nil {
			meta := map[string]any{"total": v.Total, "page": v.Page, "pageSize": v.PageSize, "nextCursor": v.NextCursor}
			b, _ := json.Marshal(meta)
			fmt.Fprintln(o.Stderr, string(b))
		}
		return nil
	default:
		return renderJSON(w, r)
	}
}

func renderCSV(w io.Writer, r Result) error {
	v, ok := r.(List)
	if !ok {
		return fmt.Errorf("csv only supports list results")
	}
	if len(v.Items) == 0 {
		return nil
	}
	cols := flatColumns(v.Items[0])
	cw := csv.NewWriter(w)
	defer cw.Flush()
	if err := cw.Write(cols); err != nil {
		return err
	}
	for _, it := range v.Items {
		row, err := flatRow(it, cols)
		if err != nil {
			return err
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	return nil
}

func renderTable(w io.Writer, r Result) error {
	v, ok := r.(List)
	if !ok {
		return renderJSON(w, r)
	}
	if len(v.Items) == 0 {
		fmt.Fprintln(w, "(empty)")
		return nil
	}
	cols := flatColumns(v.Items[0])
	widths := make([]int, len(cols))
	rows := make([][]string, 0, len(v.Items))
	for i, c := range cols {
		widths[i] = len(c)
	}
	for _, it := range v.Items {
		row, err := flatRow(it, cols)
		if err != nil {
			return err
		}
		for i, cell := range row {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
		rows = append(rows, row)
	}
	writeRow(w, cols, widths)
	sep := make([]string, len(cols))
	for i := range sep {
		sep[i] = strings.Repeat("-", widths[i])
	}
	writeRow(w, sep, widths)
	for _, row := range rows {
		writeRow(w, row, widths)
	}
	return nil
}

func writeRow(w io.Writer, cells []string, widths []int) {
	parts := make([]string, len(cells))
	for i, c := range cells {
		parts[i] = fmt.Sprintf("%-*s", widths[i], c)
	}
	fmt.Fprintln(w, strings.Join(parts, "  "))
}

func flatColumns(item any) []string {
	m := toFlatMap(item)
	cols := make([]string, 0, len(m))
	for k := range m {
		cols = append(cols, k)
	}
	sort.Strings(cols)
	return cols
}

func flatRow(item any, cols []string) ([]string, error) {
	m := toFlatMap(item)
	row := make([]string, len(cols))
	for i, c := range cols {
		row[i] = fmt.Sprintf("%v", m[c])
	}
	return row, nil
}

// toFlatMap converts an item (struct or map) into a flat map[string]any
// using JSON tags as keys. Non-flat fields are JSON-encoded.
func toFlatMap(item any) map[string]any {
	b, err := json.Marshal(item)
	if err != nil {
		return map[string]any{}
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return map[string]any{}
	}
	out := map[string]any{}
	for k, v := range m {
		switch v.(type) {
		case map[string]any, []any:
			b2, _ := json.Marshal(v)
			out[k] = string(b2)
		default:
			out[k] = v
		}
	}
	return out
}
