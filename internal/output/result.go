// Package output renders armoctl results in the formats agents and humans need.
package output

import "io"

// Result is the marker interface for the three result shapes.
type Result interface{ isResult() }

// List is the envelope every list command produces.
type List struct {
	Items      []any  `json:"items"`
	Total      int    `json:"total"`
	Page       int    `json:"page,omitempty"`
	PageSize   int    `json:"pageSize,omitempty"`
	NextCursor string `json:"nextCursor,omitempty"`
}

func (List) isResult() {}

// Get wraps a single resource object.
type Get struct {
	Object any `json:"-"`
}

func (Get) isResult() {}

// Mutation is the standard result of any mutating command.
type Mutation struct {
	Result  any  `json:"result,omitempty"`
	Changed bool `json:"changed"`
	DryRun  bool `json:"dryRun"`
}

func (Mutation) isResult() {}

// Options controls rendering.
type Options struct {
	Format        string
	Query         string
	Fields        []string
	Full          bool
	SummaryFields []string
	Stderr        io.Writer
}
