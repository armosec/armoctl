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
	// Method defaults to GET when empty. Set "POST" to send the request as POST,
	// in which case PageNum/PageSize are merged into the JSON body and Body is sent
	// as the request payload (with extra fields merged in).
	Method string
	// Body is the request body for POST list endpoints. It will be merged with
	// pagination keys before sending. nil is acceptable (becomes {}).
	Body any
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
	NextCursor string `json:"cursor"`
}

// ListPaged executes a paged list request. The ARMO API uses `pageNum` (0-based) and
// `pageSize`. Auto-pages until len(items) >= opts.Limit, opts.Limit == 0 reached,
// or the server reports fewer items than pageSize.
//
// When opts.Method == "POST", pagination params are placed in the request body
// (merged with opts.Body) rather than query params.
func (c *Client) ListPaged(ctx context.Context, path string, query url.Values, opts ListOpts) (PagedResult, error) {
	if opts.PageSize <= 0 {
		opts.PageSize = 50
	}
	if opts.Page > 0 {
		// Explicit single page mode.
		raw, err := c.fetchPage(ctx, path, query, opts.Method, opts.Body, opts.Page-1, opts.PageSize)
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
		raw, err := c.fetchPage(ctx, path, query, opts.Method, opts.Body, page, opts.PageSize)
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

func (c *Client) fetchPage(ctx context.Context, path string, query url.Values, method string, body any, pageNum, pageSize int) (rawListResponse, error) {
	if method == http.MethodPost || method == "POST" {
		// Merge pagination keys into a copy of the body map.
		merged := mergeBodyWithPaging(body, pageNum, pageSize)
		resp, err := c.Do(ctx, http.MethodPost, path, query, merged)
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
	// Default: GET with query params.
	q := cloneValues(query)
	q.Set("pageNum", strconv.Itoa(pageNum))
	q.Set("pageSize", strconv.Itoa(pageSize))
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

// mergeBodyWithPaging converts body to map[string]any (if not already), then
// injects pageNum and pageSize, returning the merged map.
func mergeBodyWithPaging(body any, pageNum, pageSize int) map[string]any {
	var m map[string]any
	if body == nil {
		m = map[string]any{}
	} else {
		// Marshal/unmarshal to get a plain map regardless of the input type.
		b, _ := json.Marshal(body)
		if err := json.Unmarshal(b, &m); err != nil || m == nil {
			m = map[string]any{}
		}
	}
	m["pageNum"] = pageNum
	m["pageSize"] = pageSize
	return m
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
