// Package imageconfig reads a container image's ENTRYPOINT/CMD from its
// registry, so armoctl can wrap containers that rely on image defaults.
package imageconfig

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"

	"github.com/armosec/armoctl/ecs/patcher"
)

// Resolver implements patcher.ImageConfigResolver against real registries.
type Resolver struct {
	keychain authn.Keychain
	nameOpts []name.Option
}

var _ patcher.ImageConfigResolver = (*Resolver)(nil)

// New returns a Resolver. ECR registries authenticate with awsCfg (nil
// disables ECR auth); every other registry uses the Docker credential store,
// then anonymous access.
func New(awsCfg *aws.Config) *Resolver {
	kcs := []authn.Keychain{}
	if awsCfg != nil {
		kcs = append(kcs, &ecrKeychain{cfg: *awsCfg, tokens: map[string]authn.AuthConfig{}})
	}
	kcs = append(kcs, authn.DefaultKeychain)
	return &Resolver{keychain: authn.NewMultiKeychain(kcs...)}
}

// Resolve returns the ENTRYPOINT/CMD of image for linux and the given ECS
// CPU architecture ("" or X86_64 → amd64, ARM64 → arm64).
func (r *Resolver) Resolve(ctx context.Context, image string, arch ecstypes.CPUArchitecture) (patcher.ImageConfig, error) {
	platform := v1.Platform{OS: "linux", Architecture: "amd64"}
	if arch == ecstypes.CPUArchitectureArm64 {
		platform.Architecture = "arm64"
	}

	ref, err := name.ParseReference(image, r.nameOpts...)
	if err != nil {
		return patcher.ImageConfig{}, fmt.Errorf("parsing image reference: %w", err)
	}
	img, err := remote.Image(ref,
		remote.WithContext(ctx),
		remote.WithAuthFromKeychain(r.keychain),
		remote.WithPlatform(platform),
	)
	if err != nil {
		return patcher.ImageConfig{}, fmt.Errorf("fetching image for %s: %w", platform, err)
	}
	cf, err := img.ConfigFile()
	if err != nil {
		return patcher.ImageConfig{}, fmt.Errorf("reading image config for %s: %w", platform, err)
	}
	// An index without the requested platform can fall back to the first
	// manifest; make sure we really got the architecture we asked for.
	if cf.OS != "" && (cf.OS != platform.OS || cf.Architecture != platform.Architecture) {
		return patcher.ImageConfig{}, fmt.Errorf("image has no %s variant (got %s/%s)", platform, cf.OS, cf.Architecture)
	}
	return patcher.ImageConfig{Entrypoint: cf.Config.Entrypoint, Cmd: cf.Config.Cmd}, nil
}

var ecrHost = regexp.MustCompile(`^\d{12}\.dkr\.ecr(?:-fips)?\.([a-z0-9-]+)\.amazonaws\.com(?:\.cn)?$`)

// ecrRegion reports whether host is a private ECR registry and its region.
func ecrRegion(host string) (string, bool) {
	m := ecrHost.FindStringSubmatch(host)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// ecrKeychain authenticates to private ECR registries with an ECR
// authorization token, cached per region.
type ecrKeychain struct {
	cfg    aws.Config
	mu     sync.Mutex
	tokens map[string]authn.AuthConfig
}

func (k *ecrKeychain) Resolve(res authn.Resource) (authn.Authenticator, error) {
	return k.ResolveContext(context.Background(), res)
}

func (k *ecrKeychain) ResolveContext(ctx context.Context, res authn.Resource) (authn.Authenticator, error) {
	region, ok := ecrRegion(res.RegistryStr())
	if !ok {
		return authn.Anonymous, nil
	}

	k.mu.Lock()
	defer k.mu.Unlock()
	if ac, ok := k.tokens[region]; ok {
		return authn.FromConfig(ac), nil
	}

	cfg := k.cfg.Copy()
	cfg.Region = region
	out, err := ecr.NewFromConfig(cfg).GetAuthorizationToken(ctx, &ecr.GetAuthorizationTokenInput{})
	if err != nil {
		return nil, fmt.Errorf("getting ECR authorization token for %s: %w", region, err)
	}
	if len(out.AuthorizationData) == 0 || out.AuthorizationData[0].AuthorizationToken == nil {
		return nil, errors.New("ECR returned no authorization token")
	}
	raw, err := base64.StdEncoding.DecodeString(*out.AuthorizationData[0].AuthorizationToken)
	if err != nil {
		return nil, fmt.Errorf("decoding ECR authorization token: %w", err)
	}
	user, pass, ok := strings.Cut(string(raw), ":")
	if !ok {
		return nil, errors.New("malformed ECR authorization token")
	}
	ac := authn.AuthConfig{Username: user, Password: pass}
	k.tokens[region] = ac
	return authn.FromConfig(ac), nil
}
