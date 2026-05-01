package operator

import (
	"testing"

	"github.com/armosec/armoctl/internal/config"
	"github.com/spf13/viper"
)

// TestECS_APIUrlDefaultUnchanged guards against accidentally changing the
// existing api-url default; ECS install code reads viper.GetString("api-url").
func TestECS_APIUrlDefaultUnchanged(t *testing.T) {
	viper.Reset()
	config.ApplyDefaults()
	if got := viper.GetString("api-url"); got != "cloud.armosec.io" {
		t.Fatalf("api-url default = %q, want cloud.armosec.io — ECS regression", got)
	}
}
