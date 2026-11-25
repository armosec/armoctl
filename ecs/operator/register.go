package operator

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/spf13/cobra"

	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/armosec/armoctl/internal/sensors"
)

// fetchSensors is the backend image lookup; a variable so tests can stub it.
var fetchSensors = sensors.Fetch

const (
	registerTimeout  = 60 * time.Second
	registerInterval = 3 * time.Second
)

// registerCluster onboards the cluster the same way the ARMO UI does:
// POST /accounts/ecs registers it (so it shows in the UI), then
// GET /accounts/ecs?clusterARN= grants the cluster's AWS account pull access
// to ARMO's private ECR images in the cluster's region.
//
// Registration is processed asynchronously, so the GET is retried while it
// answers "not found", every interval, up to timeout. A POST error (for
// example, the cluster is already registered) is returned only if the GET
// never succeeds.
func registerCluster(ctx context.Context, c *apiclient.Client, clusterARN string, timeout, interval time.Duration) error {
	postErr := c.PostJSON(ctx, "/accounts/ecs", nil, map[string]string{"clusterARN": clusterARN}, nil)

	deadline := time.Now().Add(timeout)
	query := url.Values{"clusterARN": {clusterARN}}
	for {
		var ignore map[string]any
		err := c.GetJSON(ctx, "/accounts/ecs", query, &ignore)
		if err == nil {
			return nil
		}

		var ce *clierr.Error
		if !errors.As(err, &ce) || ce.Code != clierr.CodeNotFound {
			return fmt.Errorf("registering cluster %s with ARMO: %w", clusterARN, err)
		}
		if time.Now().After(deadline) {
			if postErr != nil {
				return fmt.Errorf("cluster %s was not registered with ARMO within %s (register request failed: %v): %w", clusterARN, timeout, postErr, err)
			}
			return fmt.Errorf("cluster %s was not registered with ARMO within %s: %w", clusterARN, timeout, err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}
}

// operatorImageFor returns --operator-image when set, otherwise the operator
// image ARMO serves for region.
func operatorImageFor(cmd *cobra.Command, c *apiclient.Client, region string) (string, error) {
	if img, _ := cmd.Flags().GetString("operator-image"); img != "" {
		return img, nil
	}
	versions, err := fetchSensors(cmd.Context(), c, region)
	if err != nil {
		return "", fmt.Errorf("%w; or pass --operator-image <image>", err)
	}
	return versions.OperatorImage()
}
