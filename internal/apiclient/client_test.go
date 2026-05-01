package apiclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/armosec/armoctl/internal/clierr"
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
	_ = resp.Body.Close()
	if gotKey != "K" {
		t.Fatalf("x-api-key = %q, want K", gotKey)
	}
	if gotGUID != "G" {
		t.Fatalf("customerGUID = %q, want G", gotGUID)
	}
}

func TestGetJSON_404IsNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-request-id", "req-1")
		w.WriteHeader(404)
		_, _ = w.Write([]byte(`{"message":"nope"}`))
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	var out struct{ X int }
	err := c.GetJSON(context.Background(), "/runtime/incidents/abc", nil, &out)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	var ce *clierr.Error
	if !errors.As(err, &ce) || ce.Code != clierr.CodeNotFound {
		t.Fatalf("err = %v, want CodeNotFound", err)
	}
	if ce.RequestID != "req-1" {
		t.Fatalf("RequestID = %q, want req-1", ce.RequestID)
	}
}

func TestGetJSON_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"x":42}`))
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
	var out struct{ X int }
	if err := c.GetJSON(context.Background(), "/x", nil, &out); err != nil {
		t.Fatal(err)
	}
	if out.X != 42 {
		t.Fatalf("out.X = %d", out.X)
	}
}
