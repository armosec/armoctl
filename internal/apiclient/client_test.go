package apiclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDoInjectsAuthAndCustomerGUID(t *testing.T) {
	var gotKey, gotGUID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("x-api-key")
		gotGUID = r.URL.Query().Get("customerGUID")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	resp, err := c.Do(context.Background(), "GET", "/runtime/incidents", nil, nil)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	resp.Body.Close()
	if gotKey != "K" {
		t.Fatalf("x-api-key = %q, want K", gotKey)
	}
	if gotGUID != "G" {
		t.Fatalf("customerGUID = %q, want G", gotGUID)
	}
}
