package config

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/viper"
)

func TestDefaults_NewKey(t *testing.T) {
	viper.Reset()
	ApplyDefaults()
	if got := viper.GetString("api-base-url"); got != "api.armosec.io" {
		t.Fatalf("api-base-url default = %q, want api.armosec.io", got)
	}
}

func TestDefaults_LeavesExistingAPIURLAlone(t *testing.T) {
	// ECS / version-check still expects cloud.armosec.io as the dashboard default.
	viper.Reset()
	ApplyDefaults()
	if got := viper.GetString("api-url"); got != "cloud.armosec.io" {
		t.Fatalf("api-url default = %q, want cloud.armosec.io (ECS regression)", got)
	}
}

func TestWhoami_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "K" {
			t.Errorf("missing key")
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte("{}"))
	}))
	defer srv.Close()
	if err := Whoami(context.Background(), srv.URL, "G", "K"); err != nil {
		t.Fatal(err)
	}
}

func TestWhoami_BadKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	}))
	defer srv.Close()
	if err := Whoami(context.Background(), srv.URL, "G", "K"); err == nil {
		t.Fatal("expected error")
	}
}
