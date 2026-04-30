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
