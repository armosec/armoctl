package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/huh/v2"
	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"
	"golang.org/x/term"
)

// RequireAuth checks that customer-guid and access-key are set.
// If they are missing and stdin is a terminal, it prompts the user
// interactively and saves the values to ~/.armoctl/config.yaml.
func RequireAuth() error {
	guid := viper.GetString("customer-guid")
	key := viper.GetString("access-key")

	if guid != "" && key != "" {
		return nil
	}

	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return fmt.Errorf(`authentication required. To get your credentials:
  1. Log in to https://%s
  2. Go to Settings > Access Keys
  3. Copy your Customer GUID and Access Key

Then either:
  - Run interactively: armoctl configure
  - Set env vars: ARMO_CUSTOMER_GUID and ARMO_ACCESS_KEY
  - Save to config: ~/.armoctl/config.yaml`, viper.GetString("api-url"))
	}

	fmt.Fprintf(os.Stderr, "Credentials not found. Let's set them up.\n")
	fmt.Fprintf(os.Stderr, "You can find your credentials at https://%s → Settings → Access Keys\n\n", viper.GetString("api-url"))

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Customer GUID").
				Value(&guid).
				Validate(required("Customer GUID")),
			huh.NewInput().
				Title("Access Key").
				Value(&key).
				Validate(required("Access Key")),
		),
	)

	if err := form.Run(); err != nil {
		return fmt.Errorf("prompting for credentials: %w", err)
	}

	viper.Set("customer-guid", guid)
	viper.Set("access-key", key)

	if err := SaveConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not save config: %v\n", err)
		fmt.Fprintln(os.Stderr, "Credentials will be used for this session only.")
	} else {
		fmt.Fprintln(os.Stderr, "Credentials saved to ~/.armoctl/config.yaml")
	}
	fmt.Fprintln(os.Stderr)

	return nil
}

// PromptAllCredentials prompts for customer-guid, access-key, and api-url,
// pre-filling with current values. Use this for the "configure" command.
func PromptAllCredentials() error {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return fmt.Errorf("configure requires an interactive terminal")
	}

	guid := viper.GetString("customer-guid")
	key := viper.GetString("access-key")
	apiURL := viper.GetString("api-url")

	fmt.Fprintf(os.Stderr, "You can find your credentials at https://%s → Settings → Access Keys\n\n", apiURL)

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Customer GUID").
				Value(&guid).
				Validate(required("Customer GUID")),
			huh.NewInput().
				Title("Access Key").
				Value(&key).
				Validate(required("Access Key")),
			huh.NewInput().
				Title("API URL").
				Value(&apiURL).
				Validate(required("API URL")),
		),
	)

	if err := form.Run(); err != nil {
		return fmt.Errorf("prompting for credentials: %w", err)
	}

	viper.Set("customer-guid", guid)
	viper.Set("access-key", key)
	viper.Set("api-url", apiURL)

	if err := SaveConfig(); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	apiBase := viper.GetString("api-base-url")
	if err := Whoami(context.Background(), apiBase, guid, key); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: credentials saved but whoami ping failed: %v\n", err)
	}

	fmt.Fprintln(os.Stderr, "Configuration saved to ~/.armoctl/config.yaml")
	return nil
}

// SaveConfig merges the current credential values into ~/.armoctl/config.yaml,
// preserving any other keys already present in the file.
func SaveConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("finding home directory: %w", err)
	}

	configDir := filepath.Join(home, ".armoctl")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	if err := os.Chmod(configDir, 0o700); err != nil {
		return fmt.Errorf("setting config directory permissions: %w", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")

	existing := map[string]any{}
	if data, err := os.ReadFile(configPath); err == nil {
		_ = yaml.Unmarshal(data, &existing)
	}

	if v := viper.GetString("customer-guid"); v != "" {
		existing["customer-guid"] = v
	}
	if v := viper.GetString("access-key"); v != "" {
		existing["access-key"] = v
	}
	if v := viper.GetString("api-url"); v != "" && v != "cloud.armosec.io" {
		existing["api-url"] = v
	} else {
		delete(existing, "api-url")
	}
	if v := viper.GetString("api-base-url"); v != "" && v != "api.armosec.io" {
		existing["api-base-url"] = v
	} else {
		delete(existing, "api-base-url")
	}

	out, err := yaml.Marshal(existing)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	return os.WriteFile(configPath, out, 0o600)
}

// ApplyDefaults installs viper defaults the rest of armoctl assumes.
// Safe to call multiple times.
//
// IMPORTANT: api-url is intentionally kept at cloud.armosec.io for ECS and
// version-check compatibility. The new agent-bridge clusters use api-base-url.
func ApplyDefaults() {
	viper.SetDefault("api-url", "cloud.armosec.io")
	viper.SetDefault("api-base-url", "api.armosec.io")
}

func required(label string) func(string) error {
	return func(s string) error {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("%s is required", label)
		}
		return nil
	}
}

// Whoami pings a lightweight read endpoint to validate that
// (apiBaseURL, customerGUID, accessKey) form a working triple.
// Note: uses api-base-url (the API host), not api-url (the dashboard host).
func Whoami(ctx context.Context, apiBaseURL, customerGUID, accessKey string) error {
	c := apiclient.New(apiclient.Config{
		BaseURL:      apiBaseURL,
		AccessKey:    accessKey,
		CustomerGUID: customerGUID,
	})
	var ignore map[string]any
	return c.GetJSON(ctx, "/customerState/onboarding", nil, &ignore)
}
