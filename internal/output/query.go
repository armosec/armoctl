package output

import (
	"encoding/json"
	"fmt"

	"github.com/itchyny/gojq"
)

// applyQuery runs a gojq expression over the JSON form of input and returns the
// resulting values. Multiple results are returned as []any.
func applyQuery(input any, expr string) (any, error) {
	if expr == "" {
		return input, nil
	}
	q, err := gojq.Parse(expr)
	if err != nil {
		return nil, fmt.Errorf("parsing --query: %w", err)
	}

	// Round-trip via JSON so structs become plain maps for jq.
	b, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	var generic any
	if err := json.Unmarshal(b, &generic); err != nil {
		return nil, err
	}

	iter := q.Run(generic)
	var out []any
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, isErr := v.(error); isErr {
			return nil, fmt.Errorf("--query: %w", err)
		}
		out = append(out, v)
	}
	if len(out) == 1 {
		return out[0], nil
	}
	return out, nil
}
