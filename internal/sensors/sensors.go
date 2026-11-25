// Package sensors resolves the ECS sensor images (ptrace sidecar, operator)
// that armoctl installs. The backend is the single source of truth: its
// GET /api/v1/sensors/version returns the pinned versions and, for a region, the
// ECR repositories in ARMO's account for the current environment.
package sensors

import (
	"context"
	"fmt"
	"net/url"

	"github.com/armosec/armoctl/internal/apiclient"
)

// Images holds tagless ECR repository addresses for one region.
type Images struct {
	PtraceAgent string `json:"ptraceAgent"`
	ECSOperator string `json:"ecsOperator"`
}

// Versions is the subset of GET /sensors/version that armoctl uses.
type Versions struct {
	PtraceAgent string  `json:"ptraceAgent"`
	ECSOperator string  `json:"ecsOperator"`
	Images      *Images `json:"images,omitempty"`
}

// Fetch reads the sensor versions and the image repositories for region.
func Fetch(ctx context.Context, c *apiclient.Client, region string) (*Versions, error) {
	var v Versions
	if err := c.GetJSON(ctx, "/sensors/version", url.Values{"region": {region}}, &v); err != nil {
		return nil, fmt.Errorf("fetching sensor images from ARMO: %w", err)
	}
	return &v, nil
}

// AgentImage returns the ptrace sidecar image as repository:version.
func (v *Versions) AgentImage() (string, error) {
	var repo string
	if v.Images != nil {
		repo = v.Images.PtraceAgent
	}
	return image("ptrace agent", repo, v.PtraceAgent, "--agent-image")
}

// OperatorImage returns the ECS operator image as repository:version.
func (v *Versions) OperatorImage() (string, error) {
	var repo string
	if v.Images != nil {
		repo = v.Images.ECSOperator
	}
	return image("ECS operator", repo, v.ECSOperator, "--operator-image")
}

func image(name, repo, version, flag string) (string, error) {
	if repo == "" || version == "" {
		return "", fmt.Errorf("ARMO did not return a %s image for this region; pass %s <image> explicitly", name, flag)
	}
	return repo + ":" + version, nil
}
