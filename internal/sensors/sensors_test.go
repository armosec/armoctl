package sensors

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/armosec/armoctl/internal/apiclient"
)

const withImagesBody = `{"ptraceAgent":"v0.0.613","ecsOperator":"v0.0.32","images":{
 "ptraceAgent":"057134155174.dkr.ecr.eu-west-1.amazonaws.com/ecs-ptrace-agent",
 "ecsOperator":"057134155174.dkr.ecr.eu-west-1.amazonaws.com/ecs-operator"}}`

func serve(t *testing.T, status int, body string, check func(r *http.Request)) *apiclient.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if check != nil {
			check(r)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return apiclient.New(apiclient.Config{BaseURL: srv.URL, AccessKey: "K", CustomerGUID: "G"})
}

func TestFetch_SendsRegion(t *testing.T) {
	c := serve(t, http.StatusOK, withImagesBody, func(r *http.Request) {
		if r.URL.Path != "/api/v1/sensors/version" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("region"); got != "eu-west-1" {
			t.Errorf("region = %q", got)
		}
	})

	v, err := Fetch(context.Background(), c, "eu-west-1")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	agent, err := v.AgentImage()
	if want := "057134155174.dkr.ecr.eu-west-1.amazonaws.com/ecs-ptrace-agent:v0.0.613"; err != nil || agent != want {
		t.Errorf("AgentImage() = %q, %v; want %q", agent, err, want)
	}

	op, err := v.OperatorImage()
	if want := "057134155174.dkr.ecr.eu-west-1.amazonaws.com/ecs-operator:v0.0.32"; err != nil || op != want {
		t.Errorf("OperatorImage() = %q, %v; want %q", op, err, want)
	}
}

func TestAgentImage_NoImages_ErrorNamesFlag(t *testing.T) {
	c := serve(t, http.StatusOK, `{"ptraceAgent":"v0.0.613","ecsOperator":"v0.0.32"}`, nil)
	v, err := Fetch(context.Background(), c, "eu-west-1")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	if _, err := v.AgentImage(); err == nil || !strings.Contains(err.Error(), "--agent-image") {
		t.Errorf("AgentImage() error = %v; want it to name --agent-image", err)
	}
	if _, err := v.OperatorImage(); err == nil || !strings.Contains(err.Error(), "--operator-image") {
		t.Errorf("OperatorImage() error = %v; want it to name --operator-image", err)
	}
}

func TestAgentImage_EmptyVersion_Error(t *testing.T) {
	v := &Versions{Images: &Images{PtraceAgent: "repo/ecs-ptrace-agent"}}

	if _, err := v.AgentImage(); err == nil {
		t.Error("AgentImage() with empty version: want error")
	}
}

func TestFetch_HTTPError(t *testing.T) {
	c := serve(t, http.StatusInternalServerError, `{"message":"boom"}`, nil)

	if _, err := Fetch(context.Background(), c, "eu-west-1"); err == nil {
		t.Error("Fetch on 500: want error")
	}
}
