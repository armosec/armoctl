package imageconfig

import (
	"context"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

// newRegistry starts an in-memory registry and returns its host.
func newRegistry(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(registry.New())
	t.Cleanup(srv.Close)
	return strings.TrimPrefix(srv.URL, "http://")
}

func testImage(t *testing.T, entrypoint, cmd []string, arch string) v1.Image {
	t.Helper()
	img, err := random.Image(64, 1)
	if err != nil {
		t.Fatal(err)
	}
	cf, err := img.ConfigFile()
	if err != nil {
		t.Fatal(err)
	}
	cf = cf.DeepCopy()
	cf.OS, cf.Architecture = "linux", arch
	cf.Config.Entrypoint, cf.Config.Cmd = entrypoint, cmd
	img, err = mutate.ConfigFile(img, cf)
	if err != nil {
		t.Fatal(err)
	}
	return img
}

func pushImage(t *testing.T, ref string, img v1.Image) {
	t.Helper()
	r, err := name.ParseReference(ref, name.Insecure)
	if err != nil {
		t.Fatal(err)
	}
	if err := remote.Write(r, img); err != nil {
		t.Fatal(err)
	}
}

func pushIndex(t *testing.T, ref string, imgs map[string]v1.Image) {
	t.Helper()
	var idx v1.ImageIndex = empty.Index
	for arch, img := range imgs {
		idx = mutate.AppendManifests(idx, mutate.IndexAddendum{
			Add:        img,
			Descriptor: v1.Descriptor{Platform: &v1.Platform{OS: "linux", Architecture: arch}},
		})
	}
	r, err := name.ParseReference(ref, name.Insecure)
	if err != nil {
		t.Fatal(err)
	}
	if err := remote.WriteIndex(r, idx); err != nil {
		t.Fatal(err)
	}
}

func testResolver() *Resolver {
	r := New(nil)
	r.nameOpts = []name.Option{name.Insecure}
	return r
}

func TestResolveSingleImage(t *testing.T) {
	host := newRegistry(t)
	ref := host + "/app:1"
	pushImage(t, ref, testImage(t, []string{"/ep"}, []string{"run"}, "amd64"))

	cfg, err := testResolver().Resolve(context.Background(), ref, "")
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}
	if !slices.Equal(cfg.Entrypoint, []string{"/ep"}) || !slices.Equal(cfg.Cmd, []string{"run"}) {
		t.Errorf("Resolve() = %+v, want entrypoint [/ep] cmd [run]", cfg)
	}
}

func TestResolveMultiArch(t *testing.T) {
	host := newRegistry(t)
	ref := host + "/multi:1"
	pushIndex(t, ref, map[string]v1.Image{
		"amd64": testImage(t, nil, []string{"amd"}, "amd64"),
		"arm64": testImage(t, nil, []string{"arm"}, "arm64"),
	})

	for _, tt := range []struct {
		arch ecstypes.CPUArchitecture
		want string
	}{
		{ecstypes.CPUArchitectureArm64, "arm"},
		{"", "amd"},
		{ecstypes.CPUArchitectureX8664, "amd"},
	} {
		cfg, err := testResolver().Resolve(context.Background(), ref, tt.arch)
		if err != nil {
			t.Fatalf("Resolve(%q) error: %v", tt.arch, err)
		}
		if !slices.Equal(cfg.Cmd, []string{tt.want}) {
			t.Errorf("Resolve(%q) cmd = %v, want [%s]", tt.arch, cfg.Cmd, tt.want)
		}
	}
}

func TestResolveNoMatchingPlatform(t *testing.T) {
	host := newRegistry(t)
	ref := host + "/armonly:1"
	pushIndex(t, ref, map[string]v1.Image{"arm64": testImage(t, nil, []string{"arm"}, "arm64")})

	_, err := testResolver().Resolve(context.Background(), ref, "")
	if err == nil || !strings.Contains(err.Error(), "linux/amd64") {
		t.Fatalf("Resolve() error = %v, want one mentioning linux/amd64", err)
	}
}

func TestResolveMissingImage(t *testing.T) {
	host := newRegistry(t)
	if _, err := testResolver().Resolve(context.Background(), host+"/nope:1", ""); err == nil {
		t.Fatal("Resolve() of a missing image returned no error")
	}
}

func TestResolveInvalidReference(t *testing.T) {
	if _, err := testResolver().Resolve(context.Background(), "UPPER CASE::bad", ""); err == nil {
		t.Fatal("Resolve() of an invalid reference returned no error")
	}
}

func TestECRRegion(t *testing.T) {
	for _, tt := range []struct {
		host   string
		region string
		ok     bool
	}{
		{"123456789012.dkr.ecr.eu-west-1.amazonaws.com", "eu-west-1", true},
		{"123456789012.dkr.ecr.cn-north-1.amazonaws.com.cn", "cn-north-1", true},
		{"public.ecr.aws", "", false},
		{"index.docker.io", "", false},
		{"quay.io", "", false},
	} {
		region, ok := ecrRegion(tt.host)
		if region != tt.region || ok != tt.ok {
			t.Errorf("ecrRegion(%q) = %q, %v; want %q, %v", tt.host, region, ok, tt.region, tt.ok)
		}
	}
}
