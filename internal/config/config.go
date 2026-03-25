package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"golang.org/x/term"
	"go.yaml.in/yaml/v3"
)

// stdinReader is a shared buffered reader for all interactive prompts,
// avoiding data loss from multiple bufio.NewReader instances on os.Stdin.
var stdinReader = bufio.NewReader(os.Stdin)

// RequireAuth checks that customer-guid and access-key are set.
// If they are missing and stdin is a terminal, it prompts the user
// interactively and saves the values to ~/.armoctl/config.yaml.
func RequireAuth() error {
	guid := viper.GetString("customer-guid")
	key := viper.GetString("access-key")

	if guid != "" && key != "" {
		return nil
	}

	// If not interactive, return an error with instructions.
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return fmt.Errorf(`authentication required. To get your credentials:
  1. Log in to https://%s
  2. Go to Settings > Access Keys
  3. Copy your Customer GUID and Access Key

Then either:
  - Pass as flags: armoctl --customer-guid <GUID> --access-key <KEY> ...
  - Set env vars: ARMO_CUSTOMER_GUID and ARMO_ACCESS_KEY
  - Save to config: ~/.armoctl/config.yaml
  - Run interactively: armoctl configure`, viper.GetString("api-url"))
	}

	fmt.Fprintln(os.Stderr, "Credentials not found. Let's set them up.")
	fmt.Fprintf(os.Stderr, "You can find your credentials at https://%s → Settings → Access Keys\n\n", viper.GetString("api-url"))

	var err error
	if guid == "" {
		guid, err = promptInput("Customer GUID")
		if err != nil {
			return err
		}
	}
	if key == "" {
		key, err = promptInput("Access Key")
		if err != nil {
			return err
		}
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
// Requires an interactive terminal.
func PromptAllCredentials() error {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return fmt.Errorf("configure requires an interactive terminal")
	}

	fmt.Fprintf(os.Stderr, "You can find your credentials at https://%s → Settings → Access Keys\n\n", viper.GetString("api-url"))

	guid, err := promptInputWithDefault("Customer GUID", viper.GetString("customer-guid"))
	if err != nil {
		return err
	}
	key, err := promptInputWithDefault("Access Key", viper.GetString("access-key"))
	if err != nil {
		return err
	}
	apiURL, err := promptInputWithDefault("API URL", viper.GetString("api-url"))
	if err != nil {
		return err
	}

	viper.Set("customer-guid", guid)
	viper.Set("access-key", key)
	viper.Set("api-url", apiURL)

	if err := SaveConfig(); err != nil {
		return fmt.Errorf("saving config: %w", err)
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
	// Enforce directory permissions even if it already existed.
	if err := os.Chmod(configDir, 0o700); err != nil {
		return fmt.Errorf("setting config directory permissions: %w", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")

	// Read existing config to preserve unknown keys.
	existing := map[string]any{}
	if data, err := os.ReadFile(configPath); err == nil {
		_ = yaml.Unmarshal(data, &existing)
	}

	// Merge our keys.
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

	out, err := yaml.Marshal(existing)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	return os.WriteFile(configPath, out, 0o600)
}

func promptInput(label string) (string, error) {
	for {
		fmt.Fprintf(os.Stderr, "  %s: ", label)
		value, err := readLine(label)
		if err != nil {
			return "", err
		}
		if value != "" {
			return value, nil
		}
		fmt.Fprintf(os.Stderr, "  %s cannot be empty, please try again.\n", label)
	}
}

func promptInputWithDefault(label, current string) (string, error) {
	if current != "" {
		fmt.Fprintf(os.Stderr, "  %s [%s]: ", label, current)
	} else {
		fmt.Fprintf(os.Stderr, "  %s: ", label)
	}
	value, err := readLine(label)
	if err != nil {
		return "", err
	}
	if value == "" {
		return current, nil
	}
	return value, nil
}

func readLine(label string) (string, error) {
	value, err := stdinReader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", label, err)
	}
	return strings.TrimSpace(value), nil
}
