package patcher

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

// fakeResolver returns canned image configs and records every lookup.
type fakeResolver struct {
	configs map[string]ImageConfig
	errs    map[string]error
	calls   []string
	archs   []ecstypes.CPUArchitecture
	// fallback is returned for images not in configs.
	fallback ImageConfig
}

func (f *fakeResolver) Resolve(_ context.Context, image string, arch ecstypes.CPUArchitecture) (ImageConfig, error) {
	f.calls = append(f.calls, image)
	f.archs = append(f.archs, arch)
	if err := f.errs[image]; err != nil {
		return ImageConfig{}, err
	}
	if cfg, ok := f.configs[image]; ok {
		return cfg, nil
	}
	return f.fallback, nil
}

// testOpts gives tests that don't care about image configs a resolver that
// reports every image as having no ENTRYPOINT and CMD ["default-cmd"].
func testOpts(opts PatchOptions) PatchOptions {
	if opts.Resolver == nil {
		opts.Resolver = &fakeResolver{fallback: ImageConfig{Cmd: []string{"default-cmd"}}}
	}
	return opts
}

var testSidecar = SidecarConfig{Image: "ptrace:latest", CustomerGUID: "g", AccessKey: "k"}

func TestPatchWrapping(t *testing.T) {
	nginxEP := []string{"/docker-entrypoint.sh"}
	nginxCmd := []string{"nginx", "-g", "daemon off;"}

	tests := []struct {
		name           string
		entryPoint     []string
		command        []string
		image          ImageConfig
		wantEntryPoint []string
		wantCommand    []string
		wantLookup     bool
	}{
		{"entrypoint_and_command", []string{"/ep.sh"}, []string{"run"}, ImageConfig{}, []string{shimPath, "/ep.sh"}, []string{"run"}, false},
		{"entrypoint_only", []string{"/ep.sh"}, nil, ImageConfig{}, []string{shimPath, "/ep.sh"}, nil, false},
		{"command_only", nil, nginxCmd, ImageConfig{Entrypoint: nginxEP}, []string{shimPath, "/docker-entrypoint.sh"}, nginxCmd, true},
		{"image_defaults", nil, nil, ImageConfig{Entrypoint: nginxEP, Cmd: nginxCmd}, []string{shimPath, "/docker-entrypoint.sh"}, nginxCmd, true},
		{"image_entrypoint_only", nil, nil, ImageConfig{Entrypoint: []string{"/app"}}, []string{shimPath, "/app"}, nil, true},
		{"image_cmd_only", nil, nil, ImageConfig{Cmd: []string{"python", "app.py"}}, []string{shimPath}, []string{"python", "app.py"}, true},
		{"already_shimmed", []string{shimPath, "/ep.sh"}, []string{"run"}, ImageConfig{}, []string{shimPath, "/ep.sh"}, []string{"run"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			td := &TaskDefinition{
				Family: aws.String("f"),
				ContainerDefinitions: []ecstypes.ContainerDefinition{{
					Name:       aws.String("app"),
					Image:      aws.String("img"),
					EntryPoint: slices.Clone(tt.entryPoint),
					Command:    slices.Clone(tt.command),
				}},
			}
			res := &fakeResolver{configs: map[string]ImageConfig{"img": tt.image}}

			if err := Patch(context.Background(), td, PatchOptions{Resolver: res}, testSidecar); err != nil {
				t.Fatalf("Patch() error: %v", err)
			}

			app := containersByName(td)["app"]
			if !slices.Equal(app.EntryPoint, tt.wantEntryPoint) {
				t.Errorf("entryPoint = %q, want %q", app.EntryPoint, tt.wantEntryPoint)
			}
			if !slices.Equal(app.Command, tt.wantCommand) {
				t.Errorf("command = %q, want %q", app.Command, tt.wantCommand)
			}
			if got := len(res.calls) > 0; got != tt.wantLookup {
				t.Errorf("registry lookup = %v (calls %v), want %v", got, res.calls, tt.wantLookup)
			}
		})
	}
}

func TestPatchEmptyArgvError(t *testing.T) {
	td := &TaskDefinition{ContainerDefinitions: []ecstypes.ContainerDefinition{
		{Name: aws.String("app"), Image: aws.String("img")},
	}}
	err := Patch(context.Background(), td, PatchOptions{Resolver: &fakeResolver{}}, testSidecar)
	if err == nil {
		t.Fatal("expected error for image with no ENTRYPOINT or CMD")
	}
	for _, want := range []string{`container "app"`, `cannot read ENTRYPOINT/CMD of image "img"`, "image defines no ENTRYPOINT or CMD"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q missing %q", err, want)
		}
	}
}

func TestPatchResolverErrorLeavesTaskDefUntouched(t *testing.T) {
	td := &TaskDefinition{
		Family: aws.String("f"),
		ContainerDefinitions: []ecstypes.ContainerDefinition{
			{Name: aws.String("a"), Image: aws.String("img-a"), Command: []string{"run-a"}},
			{Name: aws.String("b"), Image: aws.String("img-b")},
		},
	}
	before := &TaskDefinition{
		Family: aws.String("f"),
		ContainerDefinitions: []ecstypes.ContainerDefinition{
			{Name: aws.String("a"), Image: aws.String("img-a"), Command: []string{"run-a"}},
			{Name: aws.String("b"), Image: aws.String("img-b")},
		},
	}
	res := &fakeResolver{errs: map[string]error{"img-b": errors.New("unauthorized")}}

	err := Patch(context.Background(), td, PatchOptions{Resolver: res}, testSidecar)
	if err == nil {
		t.Fatal("expected error")
	}
	want := `container "b": cannot read ENTRYPOINT/CMD of image "img-b": unauthorized. ` +
		`Set "entryPoint" and "command" explicitly for this container in the task definition, ` +
		`or leave it out of the targets with --container`
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q\nwant it to contain %q", err, want)
	}
	// The CLI (fang) appends the final period when it renders an error.
	if strings.HasSuffix(err.Error(), ".") {
		t.Errorf("error ends with a period; the CLI adds one and would print \"..\": %q", err)
	}
	if strings.Contains(err.Error(), `container "a"`) {
		t.Errorf("error should not mention container a: %q", err)
	}
	if !reflect.DeepEqual(td, before) {
		t.Errorf("task definition was modified on error: %+v", td)
	}
}

func TestPatchNilResolverNeedsLookup(t *testing.T) {
	td := &TaskDefinition{ContainerDefinitions: []ecstypes.ContainerDefinition{
		{Name: aws.String("app"), Image: aws.String("img"), Command: []string{"run"}},
	}}
	err := Patch(context.Background(), td, PatchOptions{}, testSidecar)
	if err == nil || !strings.Contains(err.Error(), "no image resolver configured") {
		t.Fatalf("error = %v, want one mentioning no image resolver configured", err)
	}
}

func TestPatchResolverArch(t *testing.T) {
	for _, tt := range []struct {
		name     string
		platform *ecstypes.RuntimePlatform
		want     ecstypes.CPUArchitecture
	}{
		{"arm64", &ecstypes.RuntimePlatform{CpuArchitecture: ecstypes.CPUArchitectureArm64}, ecstypes.CPUArchitectureArm64},
		{"unset", nil, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			td := &TaskDefinition{
				RuntimePlatform: tt.platform,
				ContainerDefinitions: []ecstypes.ContainerDefinition{
					{Name: aws.String("app"), Image: aws.String("img"), Command: []string{"run"}},
				},
			}
			res := &fakeResolver{}
			if err := Patch(context.Background(), td, PatchOptions{Resolver: res}, testSidecar); err != nil {
				t.Fatalf("Patch() error: %v", err)
			}
			if len(res.archs) != 1 || res.archs[0] != tt.want {
				t.Errorf("resolver archs = %v, want [%q]", res.archs, tt.want)
			}
		})
	}
}
