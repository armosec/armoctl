package operator

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/armosec/armoctl/internal/apiclient"
	"github.com/armosec/armoctl/internal/clierr"
	"github.com/armosec/armoctl/internal/sensors"
)

const testClusterARN = "arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster"

// fakeAccounts serves /api/v1/accounts/ecs: POST answers postStatus, the
// n-th GET answers getStatuses[n] (the last entry repeats).
func fakeAccounts(t *testing.T, postStatus int, postBody string, getStatuses ...int) (*apiclient.Client, *int32) {
	t.Helper()
	var gets int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/accounts/ecs" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		switch r.Method {
		case http.MethodPost:
			w.WriteHeader(postStatus)
			_, _ = w.Write([]byte(postBody))
		case http.MethodGet:
			if got := r.URL.Query().Get("clusterARN"); got != testClusterARN {
				t.Errorf("clusterARN = %q", got)
			}
			n := int(atomic.AddInt32(&gets, 1)) - 1
			if n >= len(getStatuses) {
				n = len(getStatuses) - 1
			}
			w.WriteHeader(getStatuses[n])
			_, _ = w.Write([]byte(`{"stackLink":"https://example"}`))
		}
	}))
	t.Cleanup(srv.Close)
	return apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"}), &gets
}

func TestRegister_RetriesUntilFound(t *testing.T) {
	c, gets := fakeAccounts(t, http.StatusCreated, `{}`, 404, 404, 200)

	if err := registerCluster(context.Background(), c, testClusterARN, time.Second, 5*time.Millisecond); err != nil {
		t.Fatalf("registerCluster: %v", err)
	}
	if *gets != 3 {
		t.Errorf("GETs = %d, want 3", *gets)
	}
}

func TestRegister_AlreadyRegistered(t *testing.T) {
	c, _ := fakeAccounts(t, http.StatusInternalServerError, `{"message":"already exists"}`, 200)

	if err := registerCluster(context.Background(), c, testClusterARN, time.Second, 5*time.Millisecond); err != nil {
		t.Fatalf("registerCluster: %v", err)
	}
}

func TestRegister_AuthError_NoRetry(t *testing.T) {
	c, gets := fakeAccounts(t, http.StatusForbidden, `{"message":"forbidden"}`, 403)

	err := registerCluster(context.Background(), c, testClusterARN, time.Second, 5*time.Millisecond)
	var ce *clierr.Error
	if !errors.As(err, &ce) || ce.Code != clierr.CodeAuth {
		t.Fatalf("error = %v; want clierr CodeAuth", err)
	}
	if *gets != 1 {
		t.Errorf("GETs = %d, want 1", *gets)
	}
}

func TestRegister_Timeout(t *testing.T) {
	c, _ := fakeAccounts(t, http.StatusCreated, `{}`, 404)

	err := registerCluster(context.Background(), c, testClusterARN, 30*time.Millisecond, 5*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), testClusterARN) {
		t.Fatalf("error = %v; want a timeout naming the cluster ARN", err)
	}
}

func TestInstall_OperatorImageFromBackend(t *testing.T) {
	orig := fetchSensors
	t.Cleanup(func() { fetchSensors = orig })
	var calls int
	fetchSensors = func(_ context.Context, _ *apiclient.Client, region string) (*sensors.Versions, error) {
		calls++
		if region != "us-east-1" {
			t.Errorf("region = %q, want us-east-1", region)
		}
		return &sensors.Versions{
			ECSOperator: "v0.0.32",
			Images:      &sensors.Images{ECSOperator: "057134155174.dkr.ecr.us-east-1.amazonaws.com/ecs-operator"},
		}, nil
	}

	newCmd := func(args ...string) *cobra.Command {
		cmd := &cobra.Command{Use: "install"}
		cmd.Flags().String("operator-image", "", "")
		cmd.SetContext(context.Background())
		if err := cmd.ParseFlags(args); err != nil {
			t.Fatal(err)
		}
		return cmd
	}

	img, err := operatorImageFor(newCmd(), nil, "us-east-1")
	if want := "057134155174.dkr.ecr.us-east-1.amazonaws.com/ecs-operator:v0.0.32"; err != nil || img != want {
		t.Errorf("operatorImageFor() = %q, %v; want %q", img, err, want)
	}

	calls = 0
	img, err = operatorImageFor(newCmd("--operator-image=x:1"), nil, "us-east-1")
	if err != nil || img != "x:1" || calls != 0 {
		t.Errorf("explicit flag: got %q, %v, %d backend calls; want x:1, nil, 0", img, err, calls)
	}
}
