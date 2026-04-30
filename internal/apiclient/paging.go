package apiclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// ListOpts controls paged list calls.
type ListOpts struct {
	Limit    int // total cap on items collected (0 = no cap)
	Page     int // explicit page (1-based) — disables auto-paging when >0
	PageSize int // page size to request (default 50)
}

// PagedResult is the apiclient's normalized paged response.
type PagedResult struct {
	Items      []any
	Total      int
	Page       int
	PageSize   int
	NextCursor string
}

type rawListResponse struct {
	Response   []json.RawMessage `json:"response"`
	Total      *struct {
		Value int `json:"value"`
	} `json:"total"`
	NextCursor string `json:"nextCursor"`
}

// ListPaged executes a paged GET. The ARMO API uses `pageNum` (0-based) and
// `pageSize`. Auto-pages until len(items) >= opts.Limit, opts.Limit == 0 reached,
// or the server reports fewer items than pageSize.
func (c *Client) ListPaged(ctx context.Context, path string, query url.Values, opts ListOpts) (PagedResult, error) {
	if opts.PageSize <= 0 {
		opts.PageSize = 50
	}
	if opts.Page > 0 {
		// Explicit single page mode.
		q := cloneValues(query)
		q.Set("pageNum", strconv.Itoa(opts.Page-1))
		q.Set("pageSize", strconv.Itoa(opts.PageSize))
		raw, err := c.fetchPage(ctx, path, q)
		if err != nil {
			return PagedResult{}, err
		}
		items, err := unwrapItems(raw.Response)
		if err != nil {
			return PagedResult{}, err
		}
		total := 0
		if raw.Total != nil {
			total = raw.Total.Value
		}
		return PagedResult{Items: items, Total: total, Page: opts.Page, PageSize: opts.PageSize, NextCursor: raw.NextCursor}, nil
	}

	page := 0
	out := PagedResult{Page: 1, PageSize: opts.PageSize}
	for {
		q := cloneValues(query)
		q.Set("pageNum", strconv.Itoa(page))
		q.Set("pageSize", strconv.Itoa(opts.PageSize))
		raw, err := c.fetchPage(ctx, path, q)
		if err != nil {
			return PagedResult{}, err
		}
		items, err := unwrapItems(raw.Response)
		if err != nil {
			return PagedResult{}, err
		}
		out.Items = append(out.Items, items...)
		if raw.Total != nil {
			out.Total = raw.Total.Value
		}
		out.NextCursor = raw.NextCursor

		if opts.Limit > 0 && len(out.Items) >= opts.Limit {
			out.Items = out.Items[:opts.Limit]
			return out, nil
		}
		if len(items) < opts.PageSize {
			return out, nil
		}
		page++
	}
}

func (c *Client) fetchPage(ctx context.Context, path string, q url.Values) (rawListResponse, error) {
	resp, err := c.Do(ctx, http.MethodGet, path, q, nil)
	if err != nil {
		return rawListResponse{}, err
	}
	defer resp.Body.Close()
	var raw rawListResponse
	if err := decode(resp, &raw); err != nil {
		return rawListResponse{}, err
	}
	return raw, nil
}

func unwrapItems(rows []json.RawMessage) ([]any, error) {
	out := make([]any, 0, len(rows))
	for _, r := range rows {
		var v any
		if err := json.Unmarshal(r, &v); err != nil {
			return nil, fmt.Errorf("decoding list item: %w", err)
		}
		out = append(out, v)
	}
	return out, nil
}

func cloneValues(v url.Values) url.Values {
	out := url.Values{}
	for k, vs := range v {
		for _, vv := range vs {
			out.Add(k, vv)
		}
	}
	return out
}
