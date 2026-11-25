package ecs

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/armosec/armoctl/internal/sensors"
)

// newSidecarCmd builds a command carrying the ecs persistent flags that
// sidecarConfig reads, with credentials set so RequireAuth does not prompt.
func newSidecarCmd(t *testing.T, args ...string) *cobra.Command {
	t.Helper()
	viper.Set("customer-guid", "G")
	viper.Set("access-key", "K")
	t.Cleanup(viper.Reset)

	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("agent-image", "", "")
	cmd.SetContext(context.Background())
	if err := cmd.ParseFlags(args); err != nil {
		t.Fatal(err)
	}
	return cmd
}

// stubSensors replaces the backend lookup and region detection for one test.
func stubSensors(t *testing.T, region string, regionErr error, fetch func(region string) (*sensors.Versions, error)) {
	t.Helper()
	origFetch, origRegion := fetchSensors, regionFor
	t.Cleanup(func() { fetchSensors, regionFor = origFetch, origRegion })

	regionFor = func(context.Context) (string, error) { return region, regionErr }
	fetchSensors = func(_ context.Context, _ *apiclient.Client, r string) (*sensors.Versions, error) {
		return fetch(r)
	}
}

func TestSidecarConfig_ExplicitImage_NoBackendCall(t *testing.T) {
	stubSensors(t, "", errors.New("region must not be read"), func(string) (*sensors.Versions, error) {
		t.Error("backend must not be called when --agent-image is set")
		return nil, errors.New("unexpected")
	})

	cfg, err := sidecarConfig(newSidecarCmd(t, "--agent-image=my/img:1"), "")
	if err != nil {
		t.Fatalf("sidecarConfig: %v", err)
	}
	if cfg.Image != "my/img:1" {
		t.Errorf("Image = %q, want my/img:1", cfg.Image)
	}
}

func TestSidecarConfig_ResolvesFromBackend(t *testing.T) {
	stubSensors(t, "eu-west-1", nil, func(r string) (*sensors.Versions, error) {
		if r != "eu-west-1" {
			t.Errorf("region = %q, want eu-west-1", r)
		}
		return &sensors.Versions{
			PtraceAgent: "v0.0.613",
			Images:      &sensors.Images{PtraceAgent: "057134155174.dkr.ecr.eu-west-1.amazonaws.com/ecs-ptrace-agent"},
		}, nil
	})

	cfg, err := sidecarConfig(newSidecarCmd(t), "")
	if err != nil {
		t.Fatalf("sidecarConfig: %v", err)
	}
	if want := "057134155174.dkr.ecr.eu-west-1.amazonaws.com/ecs-ptrace-agent:v0.0.613"; cfg.Image != want {
		t.Errorf("Image = %q, want %q", cfg.Image, want)
	}
}

func TestSidecarConfig_NoRegion_Error(t *testing.T) {
	stubSensors(t, "", errors.New("no region"), func(string) (*sensors.Versions, error) {
		t.Error("backend must not be called without a region")
		return nil, errors.New("unexpected")
	})

	_, err := sidecarConfig(newSidecarCmd(t), "")
	if err == nil || !strings.Contains(err.Error(), "AWS_REGION") || !strings.Contains(err.Error(), "--agent-image") {
		t.Errorf("error = %v; want it to name AWS_REGION and --agent-image", err)
	}
}

func TestSidecarConfig_NoImages_Error(t *testing.T) {
	stubSensors(t, "eu-west-1", nil, func(string) (*sensors.Versions, error) {
		return &sensors.Versions{PtraceAgent: "v0.0.613"}, nil
	})

	_, err := sidecarConfig(newSidecarCmd(t), "")
	if err == nil || !strings.Contains(err.Error(), "--agent-image") {
		t.Errorf("error = %v; want it to name --agent-image", err)
	}
}

func TestRegionFromARN(t *testing.T) {
	cases := map[string]string{
		"arn:aws:ecs:us-east-2:123456789012:task-definition/web:3": "us-east-2",
		"arn:aws:ecs:eu-west-1:123456789012:cluster/prod":          "eu-west-1",
		"task-definition.json":                                     "",
		"my-cluster":                                               "",
		"-":                                                        "",
	}
	for in, want := range cases {
		if got := regionFromARN(in); got != want {
			t.Errorf("regionFromARN(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSidecarConfig_ARNRegionWins(t *testing.T) {
	stubSensors(t, "eu-west-1", nil, func(r string) (*sensors.Versions, error) {
		if r != "us-east-2" {
			t.Errorf("region = %q, want us-east-2 from the ARN", r)
		}
		return &sensors.Versions{
			PtraceAgent: "v0.0.613",
			Images:      &sensors.Images{PtraceAgent: "057134155174.dkr.ecr.us-east-2.amazonaws.com/ecs-ptrace-agent"},
		}, nil
	})

	cfg, err := sidecarConfig(newSidecarCmd(t), "us-east-2")
	if err != nil {
		t.Fatalf("sidecarConfig: %v", err)
	}
	if want := "057134155174.dkr.ecr.us-east-2.amazonaws.com/ecs-ptrace-agent:v0.0.613"; cfg.Image != want {
		t.Errorf("Image = %q, want %q", cfg.Image, want)
	}
}
