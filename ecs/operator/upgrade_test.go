package operator

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
)

func TestUpgradeStackParams_ReplacesOnlyOperatorImage(t *testing.T) {
	params := upgradeStackParams("057134155174.dkr.ecr.us-east-1.amazonaws.com/ecs-operator:v0.0.32")

	want := map[string]bool{"Region": true, "EcsClusterName": true, "CustomerGuid": true, "AccessKey": true,
		"ApiUrl": true, "EcsOperatorImage": true, "CloudWatchLogsGroupName": true}
	if len(params) != len(want) {
		t.Fatalf("got %d params, want %d", len(params), len(want))
	}
	for _, p := range params {
		key := aws.ToString(p.ParameterKey)
		if !want[key] {
			t.Errorf("unexpected parameter %q", key)
		}
		if key == "EcsOperatorImage" {
			if got := aws.ToString(p.ParameterValue); got != "057134155174.dkr.ecr.us-east-1.amazonaws.com/ecs-operator:v0.0.32" {
				t.Errorf("EcsOperatorImage = %q", got)
			}
			continue
		}
		if !aws.ToBool(p.UsePreviousValue) || p.ParameterValue != nil {
			t.Errorf("%s must keep its previous value", key)
		}
	}
}

func TestIsNoUpdatesError(t *testing.T) {
	if !isNoUpdatesError(errors.New("operation error CloudFormation: UpdateStack, api error ValidationError: No updates are to be performed.")) {
		t.Error("want true for the CloudFormation no-op error")
	}
	if isNoUpdatesError(errors.New("ValidationError: Stack does not exist")) || isNoUpdatesError(nil) {
		t.Error("want false for other errors and nil")
	}
}

func TestUpdateOutcome(t *testing.T) {
	cases := []struct {
		status        string
		done, wantErr bool
	}{
		{"UPDATE_IN_PROGRESS", false, false},
		{"UPDATE_COMPLETE_CLEANUP_IN_PROGRESS", false, false},
		{"UPDATE_COMPLETE", true, false},
		{"UPDATE_ROLLBACK_IN_PROGRESS", false, false},
		{"UPDATE_ROLLBACK_COMPLETE", true, true},
		{"UPDATE_ROLLBACK_FAILED", true, true},
		{"UPDATE_FAILED", true, true},
	}
	for _, c := range cases {
		done, err := updateOutcome(c.status, "reason")
		if done != c.done || (err != nil) != c.wantErr {
			t.Errorf("updateOutcome(%s) = %v, %v; want done=%v err=%v", c.status, done, err, c.done, c.wantErr)
		}
	}
}
