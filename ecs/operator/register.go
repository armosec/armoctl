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
// Registration is processed asynchronously, so after a successful POST the
// GET is retried while it answers "not found", every interval, up to timeout.
// A POST auth error stops before the GET. Any other POST error is ignored if
// the GET succeeds (the cluster is already registered) and returned at once
// if the GET answers "not found".
func registerCluster(ctx context.Context, c *apiclient.Client, clusterARN string, timeout, interval time.Duration) error {
	postErr := c.PostJSON(ctx, "/accounts/ecs", nil, map[string]string{"clusterARN": clusterARN}, nil)
	if isAuthError(postErr) {
		return fmt.Errorf("registering cluster %s with ARMO: %w", clusterARN, postErr)
	}

	deadline := time.Now().Add(timeout)
	query := url.Values{"clusterARN": {clusterARN}}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
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
		// The POST failed and the cluster is not registered: nothing is
		// processing a registration, so polling cannot help.
		if postErr != nil {
			return fmt.Errorf("registering cluster %s with ARMO: %w", clusterARN, postErr)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("cluster %s was not registered with ARMO within %s: %w", clusterARN, timeout, err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func isAuthError(err error) bool {
	var ce *clierr.Error
	return errors.As(err, &ce) && ce.Code == clierr.CodeAuth
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
