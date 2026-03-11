package version

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestCheckForUpdates(t *testing.T) {
	tests := []struct {
		name       string
		current    string
		latest     *Versions
		wantUpdate bool
	}{
		{
			name:       "same version",
			current:    "v0.0.42",
			latest:     &Versions{Armoctl: "v0.0.42"},
			wantUpdate: false,
		},
		{
			name:       "new patch version available",
			current:    "v0.0.40",
			latest:     &Versions{Armoctl: "v0.0.42"},
			wantUpdate: true,
		},
		{
			name:       "new minor version available",
			current:    "v0.1.0",
			latest:     &Versions{Armoctl: "v0.2.0"},
			wantUpdate: true,
		},
		{
			name:       "new major version available",
			current:    "v1.0.0",
			latest:     &Versions{Armoctl: "v2.0.0"},
			wantUpdate: true,
		},
		{
			name:       "dev version - no update",
			current:    "dev",
			latest:     &Versions{Armoctl: "v0.0.42"},
			wantUpdate: false,
		},
		{
			name:       "empty version - no update",
			current:    "",
			latest:     &Versions{Armoctl: "v0.0.42"},
			wantUpdate: false,
		},
		{
			name:       "nil latest - no update",
			current:    "v0.0.40",
			latest:     nil,
			wantUpdate: false,
		},
		{
			name:       "current newer than latest (beta user)",
			current:    "v0.0.50",
			latest:     &Versions{Armoctl: "v0.0.42"},
			wantUpdate: false,
		},
		{
			name:       "semver comparison v0.0.9 vs v0.0.10",
			current:    "v0.0.9",
			latest:     &Versions{Armoctl: "v0.0.10"},
			wantUpdate: true,
		},
		{
			name:       "semver comparison v0.9.0 vs v0.10.0",
			current:    "v0.9.0",
			latest:     &Versions{Armoctl: "v0.10.0"},
			wantUpdate: true,
		},
		{
			name:       "pre-release version",
			current:    "v0.0.42-rc1",
			latest:     &Versions{Armoctl: "v0.0.42"},
			wantUpdate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := CheckForUpdates(tt.current, tt.latest)
			if info.HasUpdate != tt.wantUpdate {
				t.Errorf("CheckForUpdates() HasUpdate = %v, want %v", info.HasUpdate, tt.wantUpdate)
			}
			if info.ArmoCtlCurrent != tt.current {
				t.Errorf("CheckForUpdates() ArmoCtlCurrent = %v, want %v", info.ArmoCtlCurrent, tt.current)
			}
			if tt.latest != nil && info.ArmoCtlLatest != tt.latest.Armoctl {
				t.Errorf("CheckForUpdates() ArmoCtlLatest = %v, want %v", info.ArmoCtlLatest, tt.latest.Armoctl)
			}
		})
	}
}

func TestFetchLatestFromURL(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		wantVersions   *Versions
	}{
		{
			name: "successful fetch",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(Versions{
					Armoctl:     "v1.0.0",
					Operator:    "v2.0.0",
					PtraceAgent: "v3.0.0",
				})
			},
			wantErr: false,
			wantVersions: &Versions{
				Armoctl:     "v1.0.0",
				Operator:    "v2.0.0",
				PtraceAgent: "v3.0.0",
			},
		},
		{
			name: "server returns 404",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantErr: true,
		},
		{
			name: "server returns 500",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr: true,
		},
		{
			name: "server returns invalid JSON",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("not json"))
			},
			wantErr: true,
		},
		{
			name: "server returns empty JSON",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("{}"))
			},
			wantErr:      false,
			wantVersions: &Versions{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			versions, err := FetchLatestFromURL(context.Background(), server.URL)

			if tt.wantErr {
				if err == nil {
					t.Error("FetchLatestFromURL() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("FetchLatestFromURL() unexpected error: %v", err)
				return
			}

			if versions.Armoctl != tt.wantVersions.Armoctl {
				t.Errorf("Armoctl = %v, want %v", versions.Armoctl, tt.wantVersions.Armoctl)
			}
			if versions.Operator != tt.wantVersions.Operator {
				t.Errorf("Operator = %v, want %v", versions.Operator, tt.wantVersions.Operator)
			}
			if versions.PtraceAgent != tt.wantVersions.PtraceAgent {
				t.Errorf("PtraceAgent = %v, want %v", versions.PtraceAgent, tt.wantVersions.PtraceAgent)
			}
		})
	}
}

func TestFetchLatestFromURL_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow server
		time.Sleep(100 * time.Millisecond)
		json.NewEncoder(w).Encode(Versions{Armoctl: "v1.0.0"})
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := FetchLatestFromURL(ctx, server.URL)
	if err == nil {
		t.Error("FetchLatestFromURL() expected error for cancelled context, got nil")
	}
}

func TestFetchLatestFromURL_ResponseSizeLimit(t *testing.T) {
	// Create a response larger than MaxResponseSize
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Write valid JSON prefix, then padding, to exceed limit
		w.Write([]byte(`{"armoctl":"v1.0.0"`))
		// Write enough data to exceed the limit (but the limit reader will truncate)
		padding := make([]byte, MaxResponseSize+1000)
		for i := range padding {
			padding[i] = ' '
		}
		w.Write(padding)
		w.Write([]byte(`}`))
	}))
	defer server.Close()

	// This should fail because the JSON will be truncated
	_, err := FetchLatestFromURL(context.Background(), server.URL)
	if err == nil {
		t.Error("FetchLatestFromURL() expected error for oversized response")
	}
}

func TestBuildDownloadURL(t *testing.T) {
	url := buildDownloadURL()

	// Should contain the distribution URL
	if len(url) < len(DistributionURL) {
		t.Errorf("buildDownloadURL() = %v, expected longer URL", url)
	}

	// Should contain the distribution URL prefix
	if url[:len(DistributionURL)] != DistributionURL {
		t.Errorf("buildDownloadURL() should start with %v, got %v", DistributionURL, url)
	}

	// Should contain platform info
	if len(url) == len(DistributionURL) {
		t.Error("buildDownloadURL() should include platform suffix")
	}
}

func TestPadding(t *testing.T) {
	tests := []struct {
		current string
		latest  string
	}{
		{"v0.0.1", "v0.0.42"},
		{"v1.0.0", "v2.0.0"},
		{"dev", "v0.0.1"},
		{"v0.0.0", "v999.999.999"},
	}

	for _, tt := range tests {
		t.Run(tt.current+"->"+tt.latest, func(t *testing.T) {
			result := padding(tt.current, tt.latest)
			if len(result) == 0 {
				t.Error("padding() returned empty string")
			}
		})
	}
}

func TestGetAgentImage_NoCache(t *testing.T) {
	// Use a temp home directory with no cache
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	image := GetAgentImage()

	// Should fall back to "latest" tag
	expected := "015253967648.dkr.ecr.eu-north-1.amazonaws.com/ecs-ptrace-agent:latest"
	if image != expected {
		t.Errorf("GetAgentImage() = %v, want %v", image, expected)
	}
}

func TestGetAgentImage_WithCache(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Save cache with specific version
	versions := &Versions{
		Armoctl:     "v1.0.0",
		Operator:    "v2.0.0",
		PtraceAgent: "v3.0.0",
	}
	SaveCache(versions)

	image := GetAgentImage()

	// Should use cached version
	expected := "015253967648.dkr.ecr.eu-north-1.amazonaws.com/ecs-ptrace-agent:v3.0.0"
	if image != expected {
		t.Errorf("GetAgentImage() = %v, want %v", image, expected)
	}
}

func TestGetOperatorImage_NoCache(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	image := GetOperatorImage("us-east-1")

	// Should fall back to "latest" tag
	expected := "015253967648.dkr.ecr.us-east-1.amazonaws.com/ecs-operator:latest"
	if image != expected {
		t.Errorf("GetOperatorImage() = %v, want %v", image, expected)
	}
}

func TestGetOperatorImage_WithCache(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Save cache with specific version
	versions := &Versions{
		Armoctl:     "v1.0.0",
		Operator:    "v2.0.0",
		PtraceAgent: "v3.0.0",
	}
	SaveCache(versions)

	image := GetOperatorImage("eu-west-1")

	// Should use cached version with correct region
	expected := "015253967648.dkr.ecr.eu-west-1.amazonaws.com/ecs-operator:v2.0.0"
	if image != expected {
		t.Errorf("GetOperatorImage() = %v, want %v", image, expected)
	}
}

func TestGetOperatorImage_DifferentRegions(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	versions := &Versions{Operator: "v1.0.0"}
	SaveCache(versions)

	regions := []string{"us-east-1", "us-west-2", "eu-north-1", "ap-southeast-1"}
	for _, region := range regions {
		t.Run(region, func(t *testing.T) {
			image := GetOperatorImage(region)
			expectedPrefix := "015253967648.dkr.ecr." + region + ".amazonaws.com/ecs-operator:v1.0.0"
			if image != expectedPrefix {
				t.Errorf("GetOperatorImage(%s) = %v, want %v", region, image, expectedPrefix)
			}
		})
	}
}
