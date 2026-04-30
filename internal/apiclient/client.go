// Package apiclient is the shared HTTP client for the ARMO platform API.
package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/armosec/armoctl/internal/clierr"
)

type Config struct {
	BaseURL      string // e.g. "https://api.armosec.io" or "https://api.armosec.io/api/v1"
	AccessKey    string
	CustomerGUID string
	HTTPClient   *http.Client
}

type Client struct {
	cfg Config
	hc  *http.Client
}

func New(cfg Config) *Client {
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{cfg: cfg, hc: hc}
}

// Do issues a request to path (path may be absolute or relative to BaseURL).
// query params are merged onto the URL; customerGUID is injected automatically.
func (c *Client) Do(ctx context.Context, method, path string, query url.Values, body any) (*http.Response, error) {
	u, err := c.resolveURL(path, query)
	if err != nil {
		return nil, err
	}

	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		rdr = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), rdr)
	if err != nil {
		return nil, err
	}
	if c.cfg.AccessKey == "" || c.cfg.CustomerGUID == "" {
		return nil, &clierr.Error{Code: clierr.CodeAuth, Msg: "missing customer-guid or access-key", Hint: "run: armoctl configure"}
	}
	req.Header.Set("x-api-key", c.cfg.AccessKey)
	if body != nil {
		req.Header.Set("content-type", "application/json")
	}
	req.Header.Set("accept", "application/json")
	return c.hc.Do(req)
}

func (c *Client) resolveURL(path string, query url.Values) (*url.URL, error) {
	base := c.cfg.BaseURL
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		base = "https://" + base
	}
	if !strings.Contains(strings.TrimPrefix(strings.TrimPrefix(base, "https://"), "http://"), "/api/v") {
		base = strings.TrimRight(base, "/") + "/api/v1"
	}
	u, err := url.Parse(base + path)
	if err != nil {
		return nil, fmt.Errorf("resolving %s: %w", path, err)
	}
	q := u.Query()
	for k, vs := range query {
		for _, v := range vs {
			q.Add(k, v)
		}
	}
	q.Set("customerGUID", c.cfg.CustomerGUID)
	u.RawQuery = q.Encode()
	return u, nil
}
