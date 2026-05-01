package output

import (
	"encoding/json"
	"strings"
)

// effectiveFields returns the projection paths to apply, or nil for "no projection".
func effectiveFields(o Options) []string {
	switch {
	case len(o.Fields) > 0:
		return o.Fields
	case o.Full:
		return nil
	case len(o.SummaryFields) > 0:
		return o.SummaryFields
	default:
		return nil
	}
}

// project returns a copy of input keeping only the given dotted paths.
// Missing paths are silently dropped.
func project(input any, paths []string) any {
	b, err := json.Marshal(input)
	if err != nil {
		return input
	}
	var generic any
	if err := json.Unmarshal(b, &generic); err != nil {
		return input
	}
	out := map[string]any{}
	m, ok := generic.(map[string]any)
	if !ok {
		return generic
	}
	for _, p := range paths {
		copyPath(m, out, strings.Split(p, "."))
	}
	return out
}

func copyPath(src, dst map[string]any, parts []string) {
	if len(parts) == 0 {
		return
	}
	head := parts[0]
	v, ok := src[head]
	if !ok {
		return
	}
	if len(parts) == 1 {
		dst[head] = v
		return
	}
	child, ok := v.(map[string]any)
	if !ok {
		return
	}
	sub, _ := dst[head].(map[string]any)
	if sub == nil {
		sub = map[string]any{}
	}
	copyPath(child, sub, parts[1:])
	dst[head] = sub
}

func projectItems(items []any, paths []string) []any {
	if len(paths) == 0 {
		return items
	}
	out := make([]any, len(items))
	for i, it := range items {
		out[i] = project(it, paths)
	}
	return out
}
