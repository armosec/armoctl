package config

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"charm.land/huh/v2"
	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"
	"golang.org/x/term"
)

// Credentials is the input shape for non-interactive configure.
// Empty fields are left untouched (existing values preserved).
type Credentials struct {
	CustomerGUID string
	AccessKey    string
	APIBaseURL   string
	APIURL       string
}

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

	_, _ = fmt.Fprintf(os.Stderr, "Credentials not found. Let's set them up.\n")
	_, _ = fmt.Fprintf(os.Stderr, "You can find your credentials at https://%s → Settings → Access Keys\n\n", viper.GetString("api-url"))

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Customer GUID").
				Value(&guid).
				Validate(required("Customer GUID")),
			huh.NewInput().
				Title("Access Key").
				EchoMode(huh.EchoModePassword).
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
		_, _ = fmt.Fprintf(os.Stderr, "Warning: could not save config: %v\n", err)
		_, _ = fmt.Fprintln(os.Stderr, "Credentials will be used for this session only.")
	} else {
		_, _ = fmt.Fprintln(os.Stderr, "Credentials saved to ~/.armoctl/config.yaml")
	}
	_, _ = fmt.Fprintln(os.Stderr)

	return nil
}

// tenant maps a user-friendly label to the pair of hosts armoctl needs.
type tenant struct {
	label      string
	apiBaseURL string // agent-bridge API (used by every skill cluster)
	apiURL     string // legacy dashboard host (ECS / version-check)
}

const customTenantLabel = "Custom"

var tenants = []tenant{
	{"EU (default)", "api.armosec.io", "cloud.armosec.io"},
	{"US", "api.us.armosec.io", "cloud.us.armosec.io"},
}

// tenantFromHosts returns the tenant label matching the current config,
// or "" if no built-in tenant matches (custom/dev override).
func tenantFromHosts(apiBase, apiURL string) string {
	for _, t := range tenants {
		if t.apiBaseURL == apiBase && t.apiURL == apiURL {
			return t.label
		}
	}
	return ""
}

// PromptAllCredentials prompts for customer-guid, access-key, and tenant,
// pre-filling with current values for everything except the access key, which
// is treated as a secret: the input starts empty, the field description shows
// a masked preview of the saved key (if any), and an empty/whitespace
// submission keeps the existing value. Use this for the "configure" command.
func PromptAllCredentials() error {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return fmt.Errorf("configure requires an interactive terminal")
	}

	guid := viper.GetString("customer-guid")
	existingKey := viper.GetString("access-key")

	_, _ = fmt.Fprintln(os.Stderr, "Where to find your credentials:")
	_, _ = fmt.Fprintln(os.Stderr, "  • Customer GUID: ARMO Platform UI → top-right account dropdown")
	_, _ = fmt.Fprintln(os.Stderr, "  • Access Key:    https://cloud.armosec.io/settings/workspace/agent-access-keys")
	_, _ = fmt.Fprintln(os.Stderr, "                   (or https://cloud.us.armosec.io/... for US tenants)")
	_, _ = fmt.Fprintln(os.Stderr, "")

	var newKey string
	keyField := huh.NewInput().
		Title("Access Key").
		EchoMode(huh.EchoModePassword).
		Value(&newKey)
	if existingKey != "" {
		keyField = keyField.
			Description(fmt.Sprintf("Current: %s (leave empty to keep)", maskAccessKey(existingKey)))
	} else {
		keyField = keyField.Validate(required("Access Key"))
	}

	// Build tenant options for the select field.
	currentAPIBase := viper.GetString("api-base-url")
	currentAPIURL := viper.GetString("api-url")
	currentLabel := tenantFromHosts(currentAPIBase, currentAPIURL)
	if currentLabel == "" && (currentAPIBase != "api.armosec.io" || currentAPIURL != "cloud.armosec.io") {
		currentLabel = customTenantLabel
	}
	if currentLabel == "" {
		currentLabel = tenants[0].label
	}
	selectedTenant := currentLabel

	opts := make([]huh.Option[string], 0, len(tenants)+1)
	for _, t := range tenants {
		opts = append(opts, huh.NewOption(t.label, t.label))
	}
	opts = append(opts, huh.NewOption(customTenantLabel, customTenantLabel))

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Customer GUID").
				Value(&guid).
				Validate(required("Customer GUID")),
			keyField,
			huh.NewSelect[string]().
				Title("Tenant").
				Options(opts...).
				Value(&selectedTenant),
		),
	)

	if err := form.Run(); err != nil {
		return fmt.Errorf("prompting for credentials: %w", err)
	}

	key := existingKey
	if trimmed := strings.TrimSpace(newKey); trimmed != "" {
		key = trimmed
	}

	var apiBase, apiURL string
	for _, t := range tenants {
		if t.label == selectedTenant {
			apiBase = t.apiBaseURL
			apiURL = t.apiURL
			break
		}
	}

	if selectedTenant == customTenantLabel {
		apiBase = currentAPIBase
		apiURL = currentAPIURL
		customForm := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("API Base URL").
					Description("Agent-bridge API host (e.g. api-dev.armosec.io)").
					Value(&apiBase).
					Validate(required("API Base URL")),
				huh.NewInput().
					Title("Dashboard URL").
					Description("Legacy dashboard host (e.g. cloud-dev.armosec.io)").
					Value(&apiURL).
					Validate(required("Dashboard URL")),
			),
		)
		if err := customForm.Run(); err != nil {
			return fmt.Errorf("prompting for custom URLs: %w", err)
		}
	}

	return applyAndSave(Credentials{
		CustomerGUID: guid,
		AccessKey:    key,
		APIBaseURL:   apiBase,
		APIURL:       apiURL,
	}, false)
}

// SaveCredentials writes the supplied credentials non-interactively.
// Empty fields fall back to whatever is already in viper / config /
// environment, so callers can rotate just one value (e.g. the access
// key) without re-supplying everything.
//
// On completion the function pings the ARMO API. If strict is true,
// a failed ping is returned as an error (exit non-zero); otherwise it
// is logged as a warning and the config is still considered saved.
// Use strict=true for non-interactive flows where nothing else will
// surface a misconfiguration.
func SaveCredentials(creds Credentials, strict bool) error {
	return applyAndSave(creds, strict)
}

// ReadAccessKeyFromStdin reads a single line from r and trims trailing
// whitespace. Use this for the --access-key-stdin flag so secrets stay
// out of shell history and process listings.
//
// Reads only up to the first newline (or to EOF, whichever comes first)
// rather than slurping all of stdin, so a user who types the key on a
// TTY and presses Enter doesn't hang waiting for Ctrl-D. The reader is
// capped at 8 KiB to bound memory if a non-newline-terminated stream is
// piped in.
func ReadAccessKeyFromStdin(r io.Reader) (string, error) {
	br := bufio.NewReaderSize(r, 8192)
	line, err := br.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("reading access key from stdin: %w", err)
	}
	return strings.TrimRight(line, "\r\n \t"), nil
}

// applyAndSave merges non-empty fields of creds into viper, persists to
// ~/.armoctl/config.yaml, and pings the API to validate the result.
func applyAndSave(creds Credentials, strict bool) error {
	if creds.CustomerGUID != "" {
		viper.Set("customer-guid", creds.CustomerGUID)
	}
	if creds.AccessKey != "" {
		viper.Set("access-key", creds.AccessKey)
	}
	if creds.APIURL != "" {
		viper.Set("api-url", creds.APIURL)
	}
	if creds.APIBaseURL != "" {
		viper.Set("api-base-url", creds.APIBaseURL)
	}

	guid := viper.GetString("customer-guid")
	key := viper.GetString("access-key")
	if guid == "" {
		return fmt.Errorf("customer-guid is required")
	}
	if key == "" {
		return fmt.Errorf("access-key is required")
	}

	if err := SaveConfig(); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	apiBase := viper.GetString("api-base-url")
	if err := Whoami(context.Background(), apiBase, guid, key); err != nil {
		if strict {
			return fmt.Errorf("credentials saved but rejected by %s: %w", apiBase, err)
		}
		_, _ = fmt.Fprintf(os.Stderr, "Warning: credentials saved but whoami ping failed: %v\n", err)
	}

	_, _ = fmt.Fprintln(os.Stderr, "Configuration saved to ~/.armoctl/config.yaml")
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

// maskAccessKey renders a saved access key for display: short keys collapse to
// all asterisks; longer keys show their first and last 4 characters with the
// middle replaced by a fixed-width mask. Returns "" for an empty input.
func maskAccessKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return strings.Repeat("*", len(key))
	}
	return key[:4] + "****" + key[len(key)-4:]
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
