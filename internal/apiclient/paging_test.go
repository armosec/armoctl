package apiclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
)

func TestListPaged_AutoPagesUntilLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("pageNum"))
		size, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
		if size == 0 {
			size = 50
		}
		total := 7
		start := page * size
		end := start + size
		if end > total {
			end = total
		}
		items := []map[string]any{}
		for i := start; i < end; i++ {
			items = append(items, map[string]any{"i": i})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": items,
			"total":    map[string]any{"value": total},
		})
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	got, err := c.ListPaged(context.Background(), "/runtime/incidents", url.Values{}, ListOpts{Limit: 5, PageSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	if got.Total != 7 {
		t.Fatalf("total = %d, want 7", got.Total)
	}
	if len(got.Items) != 5 {
		t.Fatalf("items = %d, want 5 (capped by Limit)", len(got.Items))
	}
	if got.Items[0].(map[string]any)["i"].(float64) != 0 || got.Items[4].(map[string]any)["i"].(float64) != 4 {
		t.Fatalf("items not in order: %v", got.Items)
	}
	_ = fmt.Sprintf
}

func TestListPaged_POSTPutsPagingInBody(t *testing.T) {
	var capturedPages []map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		capturedPages = append(capturedPages, body)

		pageNum := int(body["pageNum"].(float64))
		pageSize := int(body["pageSize"].(float64))
		total := 5
		start := pageNum * pageSize
		end := start + pageSize
		if end > total {
			end = total
		}
		items := []map[string]any{}
		for i := start; i < end; i++ {
			items = append(items, map[string]any{"i": i})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": items,
			"total":    map[string]any{"value": total},
			"cursor":   "",
		})
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	got, err := c.ListPaged(context.Background(), "/runtime/incidents", url.Values{}, ListOpts{
		Limit:    4,
		PageSize: 2,
		Method:   "POST",
		Body:     map[string]any{"severity": "high"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 4 {
		t.Fatalf("items = %d, want 4", len(got.Items))
	}
	// Should have fetched 2 pages (2 items each, limit=4).
	if len(capturedPages) < 2 {
		t.Fatalf("expected >= 2 POST requests, got %d", len(capturedPages))
	}
	// Each page request must have pageNum and pageSize in the body.
	for i, pg := range capturedPages {
		if _, ok := pg["pageNum"]; !ok {
			t.Errorf("page %d body missing pageNum", i)
		}
		if _, ok := pg["pageSize"]; !ok {
			t.Errorf("page %d body missing pageSize", i)
		}
		// Extra body fields should also be present.
		if pg["severity"] != "high" {
			t.Errorf("page %d body missing severity filter", i)
		}
	}
	// pageNum should increment across pages.
	if capturedPages[0]["pageNum"].(float64) != 0 {
		t.Errorf("first page pageNum = %v, want 0", capturedPages[0]["pageNum"])
	}
	if capturedPages[1]["pageNum"].(float64) != 1 {
		t.Errorf("second page pageNum = %v, want 1", capturedPages[1]["pageNum"])
	}
}
