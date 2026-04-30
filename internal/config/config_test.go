package config

import (
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
