package patcher

import (
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

func containersByName(td *TaskDefinition) map[string]*ecstypes.ContainerDefinition {
	m := make(map[string]*ecstypes.ContainerDefinition, len(td.ContainerDefinitions))
	for i := range td.ContainerDefinitions {
		m[aws.ToString(td.ContainerDefinitions[i].Name)] = &td.ContainerDefinitions[i]
	}
	return m
}

func TestPatchMinimalTaskDef(t *testing.T) {
	td := &TaskDefinition{
		Family: aws.String("test-task"),
		ContainerDefinitions: []ecstypes.ContainerDefinition{
			{
				Name:      aws.String("app"),
				Image:     aws.String("nginx:latest"),
				Essential: aws.Bool(true),
				Command:   []string{"nginx", "-g", "daemon off;"},
			},
		},
		NetworkMode: ecstypes.NetworkModeAwsvpc,
		Cpu:         aws.String("256"),
		Memory:      aws.String("512"),
	}

	opts := PatchOptions{}
	sidecar := SidecarConfig{
		Image:        "015253967648.dkr.ecr.eu-north-1.amazonaws.com/ecs-ptrace-agent:latest",
		CustomerGUID: "test-guid-123",
		AccessKey:    "test-key-456",
	}

	if err := Patch(td, opts, sidecar); err != nil {
		t.Fatalf("Patch() returned error: %v", err)
	}

	if len(td.ContainerDefinitions) != 2 {
		t.Fatalf("expected 2 containers, got %d", len(td.ContainerDefinitions))
	}

	containers := containersByName(td)

	if _, ok := containers[volumeFixerName]; ok {
		t.Error("volume-fixer should not be present when VolumeFixer is false")
	}

	sc, ok := containers[sidecarName]
	if !ok {
		t.Fatal("sidecar-ptrace container not found")
	}
	lastIdx := len(td.ContainerDefinitions) - 1
	if aws.ToString(td.ContainerDefinitions[lastIdx].Name) != sidecarName {
		t.Errorf("expected sidecar-ptrace to be last container, got %q",
			aws.ToString(td.ContainerDefinitions[lastIdx].Name))
	}

	if sc.HealthCheck == nil || len(sc.HealthCheck.Command) < 2 ||
		sc.HealthCheck.Command[0] != "CMD-SHELL" ||
		sc.HealthCheck.Command[1] != "test -f /tmp/shared/ptrace-shim && /usr/bin/ptrace-agent --health --shim" {
		t.Errorf("sidecar health check must verify shim exists before reporting healthy, got %v", sc.HealthCheck)
	}

	if sc.LinuxParameters == nil || sc.LinuxParameters.Capabilities == nil {
		t.Fatal("sidecar-ptrace missing LinuxParameters/Capabilities")
	}
	hasPtrace := false
	for _, cap := range sc.LinuxParameters.Capabilities.Add {
		if cap == "SYS_PTRACE" {
			hasPtrace = true
			break
		}
	}
	if !hasPtrace {
		t.Error("sidecar-ptrace missing SYS_PTRACE capability")
	}

	foundGUID := false
	for _, env := range sc.Environment {
		if aws.ToString(env.Name) == "CUSTOMERGUID" && aws.ToString(env.Value) == "test-guid-123" {
			foundGUID = true
			break
		}
	}
	if !foundGUID {
		t.Error("sidecar-ptrace missing CUSTOMERGUID environment variable")
	}

	app := containers["app"]
	foundDep := false
	for _, dep := range app.DependsOn {
		if aws.ToString(dep.ContainerName) == sidecarName && dep.Condition == ecstypes.ContainerConditionHealthy {
			foundDep = true
			break
		}
	}
	if !foundDep {
		t.Error("app container missing dependsOn sidecar-ptrace HEALTHY")
	}

	expectedCmd := []string{"/tmp/shared/ptrace-shim", "nginx", "-g", "daemon off;"}
	if len(app.Command) != len(expectedCmd) {
		t.Fatalf("expected app command %v, got %v", expectedCmd, app.Command)
	}
	for i, v := range expectedCmd {
		if app.Command[i] != v {
			t.Fatalf("expected app command %v, got %v", expectedCmd, app.Command)
		}
	}

	if td.PidMode != ecstypes.PidModeTask {
		t.Errorf("expected pidMode %q, got %q", ecstypes.PidModeTask, td.PidMode)
	}

	volumeNames := make(map[string]bool)
	for _, v := range td.Volumes {
		volumeNames[aws.ToString(v.Name)] = true
	}
	if !volumeNames[sharedVolume] {
		t.Error("shared-data volume not found")
	}
	if !volumeNames[profilesVolume] {
		t.Error("profiles-data volume not found")
	}
}

func TestPatchSidecarInheritsLogConfiguration(t *testing.T) {
	logCfg := &ecstypes.LogConfiguration{
		LogDriver: ecstypes.LogDriverAwslogs,
		Options: map[string]string{
			"awslogs-group":  "/ecs/test",
			"awslogs-region": "us-east-1",
		},
	}
	td := &TaskDefinition{
		Family: aws.String("test-task"),
		ContainerDefinitions: []ecstypes.ContainerDefinition{
			{
				Name:             aws.String("app"),
				Image:            aws.String("nginx:latest"),
				Essential:        aws.Bool(true),
				LogConfiguration: logCfg,
			},
		},
	}

	if err := Patch(td, PatchOptions{}, SidecarConfig{Image: "ptrace:latest"}); err != nil {
		t.Fatalf("Patch() error: %v", err)
	}

	sc := containersByName(td)[sidecarName]
	if sc == nil {
		t.Fatal("sidecar-ptrace not found")
	}
	if sc.LogConfiguration == nil {
		t.Fatal("sidecar-ptrace missing LogConfiguration")
	}
	if sc.LogConfiguration.LogDriver != ecstypes.LogDriverAwslogs {
		t.Errorf("expected logDriver %q, got %q", ecstypes.LogDriverAwslogs, sc.LogConfiguration.LogDriver)
	}
	if sc.LogConfiguration.Options["awslogs-group"] != "/ecs/test" {
		t.Errorf("expected awslogs-group %q, got %q", "/ecs/test", sc.LogConfiguration.Options["awslogs-group"])
	}
	if sc.LogConfiguration.Options["awslogs-stream-prefix"] != sidecarName {
		t.Errorf("expected awslogs-stream-prefix %q, got %q", sidecarName, sc.LogConfiguration.Options["awslogs-stream-prefix"])
	}
	// Verify original container's log config is not mutated.
	app := td.ContainerDefinitions[0]
	if app.LogConfiguration.Options["awslogs-stream-prefix"] != "" {
		t.Errorf("original container LogConfiguration was mutated")
	}
}

func TestPatchSelectiveContainers(t *testing.T) {
	td := &TaskDefinition{
		Family: aws.String("multi-container-task"),
		ContainerDefinitions: []ecstypes.ContainerDefinition{
			{
				Name:      aws.String("web"),
				Image:     aws.String("nginx:latest"),
				Essential: aws.Bool(true),
				Command:   []string{"nginx", "-g", "daemon off;"},
			},
			{
				Name:      aws.String("worker"),
				Image:     aws.String("my-worker:latest"),
				Essential: aws.Bool(true),
				Command:   []string{"python", "worker.py"},
			},
		},
		NetworkMode: ecstypes.NetworkModeAwsvpc,
		Cpu:         aws.String("512"),
		Memory:      aws.String("1024"),
	}

	opts := PatchOptions{
		Containers: []string{"web"},
	}
	sidecar := SidecarConfig{
		Image:        "015253967648.dkr.ecr.eu-north-1.amazonaws.com/ecs-ptrace-agent:latest",
		CustomerGUID: "test-guid",
		AccessKey:    "test-key",
	}

	if err := Patch(td, opts, sidecar); err != nil {
		t.Fatalf("Patch() returned error: %v", err)
	}

	containers := containersByName(td)

	web := containers["web"]
	if web == nil {
		t.Fatal("web container not found")
	}
	webHasDep := false
	for _, dep := range web.DependsOn {
		if aws.ToString(dep.ContainerName) == sidecarName {
			webHasDep = true
			break
		}
	}
	if !webHasDep {
		t.Error("web container should have dependsOn sidecar-ptrace")
	}

	worker := containers["worker"]
	if worker == nil {
		t.Fatal("worker container not found")
	}
	for _, dep := range worker.DependsOn {
		if aws.ToString(dep.ContainerName) == sidecarName {
			t.Error("worker container should NOT have dependsOn sidecar-ptrace")
			break
		}
	}

	if len(worker.Command) > 0 && worker.Command[0] == "/tmp/shared/ptrace-shim" {
		t.Error("worker command should NOT be wrapped with ptrace-shim")
	}
}

func TestPatchAlreadyPatched(t *testing.T) {
	td := &TaskDefinition{
		Family: aws.String("already-patched-task"),
		ContainerDefinitions: []ecstypes.ContainerDefinition{
			{
				Name:  aws.String("app"),
				Image: aws.String("nginx:latest"),
			},
			{
				Name:  aws.String(sidecarName),
				Image: aws.String("015253967648.dkr.ecr.eu-north-1.amazonaws.com/ecs-ptrace-agent:latest"),
			},
		},
	}

	opts := PatchOptions{}
	sidecar := SidecarConfig{
		Image:        "015253967648.dkr.ecr.eu-north-1.amazonaws.com/ecs-ptrace-agent:latest",
		CustomerGUID: "test-guid",
		AccessKey:    "test-key",
	}

	err := Patch(td, opts, sidecar)
	if err == nil {
		t.Fatal("expected error for already-patched task definition, got nil")
	}
	if !strings.Contains(err.Error(), "already patched") {
		t.Errorf("expected 'already patched' in error message, got: %v", err)
	}
}

func TestPatchEmptyCommand(t *testing.T) {
	td := &TaskDefinition{
		Family: aws.String("no-command-task"),
		ContainerDefinitions: []ecstypes.ContainerDefinition{
			{
				Name:      aws.String("app"),
				Image:     aws.String("nginx:latest"),
				Essential: aws.Bool(true),
				// No Command — relies on image ENTRYPOINT/CMD.
			},
		},
	}

	opts := PatchOptions{}
	sidecar := SidecarConfig{
		Image:        "015253967648.dkr.ecr.eu-north-1.amazonaws.com/ecs-ptrace-agent:latest",
		CustomerGUID: "test-guid",
		AccessKey:    "test-key",
	}

	if err := Patch(td, opts, sidecar); err != nil {
		t.Fatalf("Patch() returned error: %v", err)
	}

	containers := containersByName(td)

	app := containers["app"]
	if app == nil {
		t.Fatal("app container not found")
	}

	if len(app.Command) > 0 {
		t.Errorf("expected empty command for container without original command, got %v", app.Command)
	}

	foundDep := false
	for _, dep := range app.DependsOn {
		if aws.ToString(dep.ContainerName) == sidecarName && dep.Condition == ecstypes.ContainerConditionHealthy {
			foundDep = true
			break
		}
	}
	if !foundDep {
		t.Error("app container missing dependsOn sidecar-ptrace HEALTHY")
	}

	hasSharedMount := false
	for _, mp := range app.MountPoints {
		if aws.ToString(mp.SourceVolume) == sharedVolume {
			hasSharedMount = true
			break
		}
	}
	if !hasSharedMount {
		t.Error("app container missing shared-data mount")
	}
}

func TestPatchPreservesPrivileged(t *testing.T) {
	td := &TaskDefinition{
		Family: aws.String("privileged-task"),
		ContainerDefinitions: []ecstypes.ContainerDefinition{
			{
				Name:       aws.String("app"),
				Image:      aws.String("nginx:latest"),
				Essential:  aws.Bool(true),
				Privileged: aws.Bool(true),
				Command:    []string{"nginx"},
			},
		},
	}

	opts := PatchOptions{}
	sidecar := SidecarConfig{
		Image:        "015253967648.dkr.ecr.eu-north-1.amazonaws.com/ecs-ptrace-agent:latest",
		CustomerGUID: "test-guid",
		AccessKey:    "test-key",
	}

	if err := Patch(td, opts, sidecar); err != nil {
		t.Fatalf("Patch() returned error: %v", err)
	}

	containers := containersByName(td)

	app := containers["app"]
	if app == nil {
		t.Fatal("app container not found")
	}
	if app.Privileged == nil || !*app.Privileged {
		t.Error("app container should preserve privileged=true")
	}
}

func TestPatchWithVolumeFixer(t *testing.T) {
	td := &TaskDefinition{
		Family: aws.String("vf-task"),
		ContainerDefinitions: []ecstypes.ContainerDefinition{
			{
				Name:    aws.String("app"),
				Image:   aws.String("nginx:latest"),
				Command: []string{"nginx"},
			},
		},
	}

	opts := PatchOptions{
		VolumeFixer: true,
	}
	sidecar := SidecarConfig{
		Image:        "015253967648.dkr.ecr.eu-north-1.amazonaws.com/ecs-ptrace-agent:latest",
		CustomerGUID: "test-guid",
		AccessKey:    "test-key",
	}

	if err := Patch(td, opts, sidecar); err != nil {
		t.Fatalf("Patch() returned error: %v", err)
	}

	if len(td.ContainerDefinitions) != 3 {
		t.Fatalf("expected 3 containers, got %d", len(td.ContainerDefinitions))
	}

	if aws.ToString(td.ContainerDefinitions[0].Name) != volumeFixerName {
		t.Errorf("expected volume-fixer to be first container, got %q",
			aws.ToString(td.ContainerDefinitions[0].Name))
	}

	containers := containersByName(td)
	sc := containers[sidecarName]
	if sc == nil {
		t.Fatal("sidecar-ptrace not found")
	}
	foundDep := false
	for _, dep := range sc.DependsOn {
		if aws.ToString(dep.ContainerName) == volumeFixerName {
			foundDep = true
			break
		}
	}
	if !foundDep {
		t.Error("sidecar should depend on volume-fixer when VolumeFixer is true")
	}
}

func TestPatchSidecarLogConfigSkipsContainersWithoutLogConfig(t *testing.T) {
	logCfg := &ecstypes.LogConfiguration{
		LogDriver: ecstypes.LogDriverAwslogs,
		Options: map[string]string{
			"awslogs-group":         "/ecs/test",
			"awslogs-region":        "us-east-1",
			"awslogs-stream-prefix": "worker",
		},
	}
	td := &TaskDefinition{
		Family: aws.String("mixed-log-task"),
		ContainerDefinitions: []ecstypes.ContainerDefinition{
			{
				Name:      aws.String("web"),
				Image:     aws.String("nginx:latest"),
				Essential: aws.Bool(true),
				Command:   []string{"nginx"},
				// No LogConfiguration
			},
			{
				Name:             aws.String("worker"),
				Image:            aws.String("python:3.11"),
				Essential:        aws.Bool(true),
				Command:          []string{"python", "worker.py"},
				LogConfiguration: logCfg,
			},
		},
	}

	if err := Patch(td, PatchOptions{}, SidecarConfig{Image: "ptrace:latest"}); err != nil {
		t.Fatalf("Patch() error: %v", err)
	}

	sc := containersByName(td)[sidecarName]
	if sc == nil {
		t.Fatal("sidecar-ptrace not found")
	}
	if sc.LogConfiguration == nil {
		t.Fatal("sidecar should inherit LogConfiguration from worker, got nil")
	}
	if sc.LogConfiguration.LogDriver != ecstypes.LogDriverAwslogs {
		t.Errorf("expected logDriver %q, got %q", ecstypes.LogDriverAwslogs, sc.LogConfiguration.LogDriver)
	}
	if sc.LogConfiguration.Options["awslogs-group"] != "/ecs/test" {
		t.Errorf("expected awslogs-group %q, got %q", "/ecs/test", sc.LogConfiguration.Options["awslogs-group"])
	}
	if sc.LogConfiguration.Options["awslogs-stream-prefix"] != sidecarName {
		t.Errorf("expected awslogs-stream-prefix %q, got %q", sidecarName, sc.LogConfiguration.Options["awslogs-stream-prefix"])
	}
}

func TestPatchSidecarNoLogConfigWhenNoTargetHasOne(t *testing.T) {
	td := &TaskDefinition{
		Family: aws.String("no-log-task"),
		ContainerDefinitions: []ecstypes.ContainerDefinition{
			{
				Name:    aws.String("app"),
				Image:   aws.String("nginx:latest"),
				Command: []string{"nginx"},
			},
		},
	}

	if err := Patch(td, PatchOptions{}, SidecarConfig{Image: "ptrace:latest"}); err != nil {
		t.Fatalf("Patch() error: %v", err)
	}

	sc := containersByName(td)[sidecarName]
	if sc == nil {
		t.Fatal("sidecar-ptrace not found")
	}
	if sc.LogConfiguration != nil {
		t.Errorf("expected nil LogConfiguration when no target has one, got %+v", sc.LogConfiguration)
	}
}

func TestWrapCommand(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "simple command",
			input:    []string{"nginx"},
			expected: []string{"/tmp/shared/ptrace-shim", "nginx"},
		},
		{
			name:     "command with args",
			input:    []string{"nginx", "-g", "daemon off;"},
			expected: []string{"/tmp/shared/ptrace-shim", "nginx", "-g", "daemon off;"},
		},
		{
			name:     "sh -c with quotes",
			input:    []string{"sh", "-c", `echo "hello"`},
			expected: []string{"/tmp/shared/ptrace-shim", "sh", "-c", `echo "hello"`},
		},
		{
			name:     "single element",
			input:    []string{"python"},
			expected: []string{"/tmp/shared/ptrace-shim", "python"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wrapCommand(tt.input)
			if len(got) != len(tt.expected) {
				t.Fatalf("wrapCommand(%v) = %v, want %v", tt.input, got, tt.expected)
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Fatalf("wrapCommand(%v) = %v, want %v", tt.input, got, tt.expected)
				}
			}
		})
	}
}

func TestMarshalTaskDefCamelCase(t *testing.T) {
	td := &TaskDefinition{
		Family: aws.String("test"),
		ContainerDefinitions: []ecstypes.ContainerDefinition{
			{
				Name:  aws.String("app"),
				Image: aws.String("nginx"),
			},
		},
		PidMode: ecstypes.PidModeTask,
		Tags: []ecstypes.Tag{
			{Key: aws.String("env"), Value: aws.String("test")},
		},
	}

	data, err := MarshalTaskDef(td)
	if err != nil {
		t.Fatalf("MarshalTaskDef() error: %v", err)
	}

	output := string(data)

	// Verify camelCase keys in output.
	if !strings.Contains(output, `"family"`) {
		t.Error("expected camelCase key 'family' in output")
	}
	if !strings.Contains(output, `"containerDefinitions"`) {
		t.Error("expected camelCase key 'containerDefinitions' in output")
	}
	if !strings.Contains(output, `"pidMode"`) {
		t.Error("expected camelCase key 'pidMode' in output")
	}
	if !strings.Contains(output, `"tags"`) {
		t.Error("expected camelCase key 'tags' in output")
	}

	// Verify NO PascalCase keys.
	if strings.Contains(output, `"Family"`) {
		t.Error("unexpected PascalCase key 'Family' in output")
	}
	if strings.Contains(output, `"ContainerDefinitions"`) {
		t.Error("unexpected PascalCase key 'ContainerDefinitions' in output")
	}
}

func TestMarshalPreservesDockerLabelsKeys(t *testing.T) {
	td := &TaskDefinition{
		Family: aws.String("labels-task"),
		ContainerDefinitions: []ecstypes.ContainerDefinition{
			{
				Name:  aws.String("app"),
				Image: aws.String("nginx"),
				DockerLabels: map[string]string{
					"com.example.MyLabel": "value1",
					"UPPERCASE_KEY":       "value2",
				},
			},
		},
	}

	data, err := MarshalTaskDef(td)
	if err != nil {
		t.Fatalf("MarshalTaskDef() error: %v", err)
	}

	output := string(data)

	// DockerLabels keys must be preserved exactly — NOT converted to camelCase.
	if !strings.Contains(output, `"com.example.MyLabel"`) {
		t.Errorf("DockerLabels key 'com.example.MyLabel' was corrupted, got:\n%s", output)
	}
	if !strings.Contains(output, `"UPPERCASE_KEY"`) {
		t.Errorf("DockerLabels key 'UPPERCASE_KEY' was corrupted, got:\n%s", output)
	}

	// The struct field itself should be camelCase.
	if !strings.Contains(output, `"dockerLabels"`) {
		t.Error("expected struct field key 'dockerLabels' in output")
	}
}

func TestMarshalPreservesEmptyStringValues(t *testing.T) {
	td := &TaskDefinition{
		Family: aws.String("empty-val-task"),
		ContainerDefinitions: []ecstypes.ContainerDefinition{
			{
				Name:  aws.String("app"),
				Image: aws.String("nginx"),
				Environment: []ecstypes.KeyValuePair{
					{Name: aws.String("EMPTY_VAR"), Value: aws.String("")},
					{Name: aws.String("SET_VAR"), Value: aws.String("hello")},
				},
			},
		},
	}

	data, err := MarshalTaskDef(td)
	if err != nil {
		t.Fatalf("MarshalTaskDef() error: %v", err)
	}

	output := string(data)

	// Empty string *string values must be preserved (not stripped).
	if !strings.Contains(output, `"EMPTY_VAR"`) {
		t.Errorf("env var 'EMPTY_VAR' was stripped from output:\n%s", output)
	}

	// Verify the value key is present (even though value is "").
	// Unmarshal the output back and check.
	td2, err := UnmarshalTaskDef(data)
	if err != nil {
		t.Fatalf("UnmarshalTaskDef() round-trip error: %v", err)
	}

	found := false
	for _, env := range td2.ContainerDefinitions[0].Environment {
		if aws.ToString(env.Name) == "EMPTY_VAR" {
			found = true
			if env.Value == nil {
				t.Error("EMPTY_VAR value was nil after round-trip, expected empty string")
			} else if *env.Value != "" {
				t.Errorf("EMPTY_VAR value was %q after round-trip, expected empty string", *env.Value)
			}
		}
	}
	if !found {
		t.Error("EMPTY_VAR env var was lost during round-trip")
	}
}

func TestMarshalRoundTrip(t *testing.T) {
	input := `{
		"family": "round-trip-test",
		"containerDefinitions": [
			{
				"name": "app",
				"image": "nginx:latest",
				"privileged": true,
				"essential": true,
				"memory": 512,
				"command": ["nginx"],
				"dockerLabels": {
					"com.example.Team": "platform",
					"KEEP_THIS_CASE": "yes"
				},
				"environment": [
					{"name": "EMPTY", "value": ""},
					{"name": "SET", "value": "hello"}
				],
				"logConfiguration": {
					"logDriver": "awslogs",
					"options": {
						"awslogs-group": "/ecs/test",
						"awslogs-region": "us-east-1"
					}
				}
			}
		],
		"networkMode": "bridge",
		"pidMode": "host",
		"tags": [
			{"key": "Purpose", "value": "Test"}
		]
	}`

	// Unmarshal → Patch → Marshal round-trip.
	td, err := UnmarshalTaskDef([]byte(input))
	if err != nil {
		t.Fatalf("UnmarshalTaskDef() error: %v", err)
	}

	opts := PatchOptions{}
	sidecar := SidecarConfig{
		Image:        "015253967648.dkr.ecr.eu-north-1.amazonaws.com/ecs-ptrace-agent:latest",
		CustomerGUID: "test-guid",
		AccessKey:    "test-key",
	}
	if err := Patch(td, opts, sidecar); err != nil {
		t.Fatalf("Patch() error: %v", err)
	}

	data, err := MarshalTaskDef(td)
	if err != nil {
		t.Fatalf("MarshalTaskDef() error: %v", err)
	}

	output := string(data)

	// Verify DockerLabels keys preserved.
	if !strings.Contains(output, `"com.example.Team"`) {
		t.Error("DockerLabels key 'com.example.Team' was corrupted in round-trip")
	}
	if !strings.Contains(output, `"KEEP_THIS_CASE"`) {
		t.Error("DockerLabels key 'KEEP_THIS_CASE' was corrupted in round-trip")
	}

	// Verify LogConfiguration options keys preserved.
	if !strings.Contains(output, `"awslogs-group"`) {
		t.Error("LogConfiguration option 'awslogs-group' was corrupted in round-trip")
	}

	// Verify tags preserved.
	if !strings.Contains(output, `"tags"`) {
		t.Error("tags field lost in round-trip")
	}

	// Verify privileged preserved.
	if !strings.Contains(output, `"privileged": true`) {
		t.Error("privileged field lost in round-trip")
	}
}

func TestUnmarshalTaskDefCamelCase(t *testing.T) {
	input := `{
		"family": "test-task",
		"containerDefinitions": [
			{
				"name": "app",
				"image": "nginx:latest",
				"privileged": true,
				"essential": true,
				"command": ["nginx"],
				"memory": 512
			}
		],
		"networkMode": "bridge",
		"pidMode": "host",
		"tags": [
			{"key": "Purpose", "value": "Test"}
		]
	}`

	td, err := UnmarshalTaskDef([]byte(input))
	if err != nil {
		t.Fatalf("UnmarshalTaskDef() error: %v", err)
	}

	if aws.ToString(td.Family) != "test-task" {
		t.Errorf("expected family 'test-task', got %q", aws.ToString(td.Family))
	}
	if len(td.ContainerDefinitions) != 1 {
		t.Fatalf("expected 1 container, got %d", len(td.ContainerDefinitions))
	}

	c := td.ContainerDefinitions[0]
	if aws.ToString(c.Name) != "app" {
		t.Errorf("expected container name 'app', got %q", aws.ToString(c.Name))
	}
	if c.Privileged == nil || !*c.Privileged {
		t.Error("expected privileged=true to be preserved")
	}
	if string(td.NetworkMode) != "bridge" {
		t.Errorf("expected networkMode 'bridge', got %q", td.NetworkMode)
	}
	if string(td.PidMode) != "host" {
		t.Errorf("expected pidMode 'host', got %q", td.PidMode)
	}
	if len(td.Tags) != 1 {
		t.Fatalf("expected 1 tag, got %d", len(td.Tags))
	}
	if aws.ToString(td.Tags[0].Key) != "Purpose" {
		t.Errorf("expected tag key 'Purpose', got %q", aws.ToString(td.Tags[0].Key))
	}
}
