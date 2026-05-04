package config

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func TestReadAccessKeyFromStdin(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"no_newline", "abc", "abc"},
		{"trailing_lf", "abc\n", "abc"},
		{"trailing_crlf", "abc\r\n", "abc"},
		{"trailing_spaces", "  abc  \n", "  abc"}, // leading whitespace preserved
		{"empty", "", ""},
		// Multi-line input: only the first line is read so subsequent
		// piped data (rare but possible) doesn't bleed into the key.
		{"multiline_takes_first", "abc\nignored\n", "abc"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ReadAccessKeyFromStdin(strings.NewReader(tc.in))
			if err != nil {
				t.Fatalf("ReadAccessKeyFromStdin(%q): unexpected error %v", tc.in, err)
			}
			if got != tc.want {
				t.Errorf("ReadAccessKeyFromStdin(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestSaveCredentials_PersistsAndPings(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "K" {
			t.Errorf("missing key")
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte("{}"))
	}))
	defer srv.Close()

	dir := t.TempDir()
	t.Setenv("HOME", dir)

	viper.Reset()
	ApplyDefaults()

	err := SaveCredentials(Credentials{
		CustomerGUID: "G",
		AccessKey:    "K",
		APIBaseURL:   srv.URL,
	}, true)
	if err != nil {
		t.Fatalf("SaveCredentials: %v", err)
	}

	if got := viper.GetString("customer-guid"); got != "G" {
		t.Errorf("customer-guid = %q, want G", got)
	}
	if got := viper.GetString("access-key"); got != "K" {
		t.Errorf("access-key = %q, want K", got)
	}
	cfgPath := filepath.Join(dir, ".armoctl", "config.yaml")
	if _, err := os.ReadFile(cfgPath); err != nil {
		t.Errorf("config not persisted at %s: %v", cfgPath, err)
	}
}

func TestSaveCredentials_StrictReturnsErrorOnBadAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(401)
	}))
	defer srv.Close()

	t.Setenv("HOME", t.TempDir())
	viper.Reset()
	ApplyDefaults()

	err := SaveCredentials(Credentials{
		CustomerGUID: "G",
		AccessKey:    "BAD",
		APIBaseURL:   srv.URL,
	}, true)
	if err == nil {
		t.Fatal("expected strict mode to surface whoami failure as error")
	}
}

func TestSaveCredentials_RequiresGUIDAndKey(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	viper.Reset()
	ApplyDefaults()

	if err := SaveCredentials(Credentials{AccessKey: "K"}, true); err == nil {
		t.Error("expected error when customer-guid is missing")
	}
	viper.Reset()
	ApplyDefaults()
	if err := SaveCredentials(Credentials{CustomerGUID: "G"}, true); err == nil {
		t.Error("expected error when access-key is missing")
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
