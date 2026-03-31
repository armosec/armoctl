package patcher

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

const (
	sidecarName       = "sidecar-ptrace"
	volumeFixerName   = "volume-fixer"
	volumeFixerImage  = "alpine:3.21"
	sharedVolume      = "shared-data"
	profilesVolume    = "profiles-data"
	sharedMountPath   = "/tmp/shared"
	profilesMountPath = "/profiles"
)

// Patch modifies the given TaskDefinition in place, injecting the ARMO ptrace
// sidecar and volume-fixer init container. Target containers are wrapped to
// launch through the ptrace shim.
func Patch(td *TaskDefinition, opts PatchOptions, sidecar SidecarConfig) error {
	for _, c := range td.ContainerDefinitions {
		if aws.ToString(c.Name) == sidecarName {
			return fmt.Errorf("task definition is already patched: container %q exists", sidecarName)
		}
	}

	targets, err := resolveTargets(td, opts.Containers)
	if err != nil {
		return err
	}

	addVolumeIfMissing(td, sharedVolume)
	addVolumeIfMissing(td, profilesVolume)

	td.PidMode = ecstypes.PidModeTask

	// Inherit log configuration from the first target container so sidecar
	// logs are shipped to the same log group. For awslogs, override the
	// stream prefix so sidecar streams are distinguishable from app streams.
	logConfig := sidecarLogConfig(td, targets)

	sc := ecstypes.ContainerDefinition{
		Name:             aws.String(sidecarName),
		Image:            aws.String(sidecar.Image),
		Essential:        aws.Bool(true),
		Command:          []string{"--shim"},
		LogConfiguration: logConfig,
		Environment: []ecstypes.KeyValuePair{
			{Name: aws.String("KS_LOGGER_LEVEL"), Value: aws.String("info")},
			{Name: aws.String("HEALTH_REPORT_INTERVAL"), Value: aws.String("5s")},
			{Name: aws.String("X_API_KEY"), Value: aws.String(sidecar.AccessKey)},
			{Name: aws.String("CUSTOMERGUID"), Value: aws.String(sidecar.CustomerGUID)},
		},
		MountPoints: []ecstypes.MountPoint{
			{SourceVolume: aws.String(profilesVolume), ContainerPath: aws.String(profilesMountPath)},
			{SourceVolume: aws.String(sharedVolume), ContainerPath: aws.String(sharedMountPath)},
		},
		LinuxParameters: &ecstypes.LinuxParameters{
			Capabilities: &ecstypes.KernelCapabilities{
				Add: []string{"SYS_PTRACE"},
			},
		},
		HealthCheck: &ecstypes.HealthCheck{
			Command:     []string{"CMD", "/usr/bin/ptrace-agent", "--health", " --shim"},
			Interval:    aws.Int32(5),
			Timeout:     aws.Int32(2),
			Retries:     aws.Int32(3),
			StartPeriod: aws.Int32(10),
		},
	}

	if opts.VolumeFixer {
		volumeFixer := ecstypes.ContainerDefinition{
			Name:      aws.String(volumeFixerName),
			Image:     aws.String(volumeFixerImage),
			Essential: aws.Bool(false),
			Command:   []string{"sh", "-c", "chmod -R 777 /tmp/shared && chown -R 1000:1000 /tmp/shared"},
			MountPoints: []ecstypes.MountPoint{
				{SourceVolume: aws.String(sharedVolume), ContainerPath: aws.String(sharedMountPath)},
			},
		}
		sc.DependsOn = []ecstypes.ContainerDependency{
			{ContainerName: aws.String(volumeFixerName), Condition: ecstypes.ContainerConditionSuccess},
		}
		td.ContainerDefinitions = append([]ecstypes.ContainerDefinition{volumeFixer}, td.ContainerDefinitions...)
	}

	for i := range td.ContainerDefinitions {
		name := aws.ToString(td.ContainerDefinitions[i].Name)
		if !targets[name] {
			continue
		}

		c := &td.ContainerDefinitions[i]

		// Skip command wrapping if the container has no command (relies on image ENTRYPOINT/CMD).
		if len(c.Command) > 0 {
			c.Command = wrapCommand(c.Command)
		}

		hasSharedMount := false
		for _, mp := range c.MountPoints {
			if aws.ToString(mp.SourceVolume) == sharedVolume {
				hasSharedMount = true
				break
			}
		}
		if !hasSharedMount {
			c.MountPoints = append(c.MountPoints, ecstypes.MountPoint{
				SourceVolume:  aws.String(sharedVolume),
				ContainerPath: aws.String(sharedMountPath),
			})
		}

		c.DependsOn = append(c.DependsOn, ecstypes.ContainerDependency{
			ContainerName: aws.String(sidecarName),
			Condition:     ecstypes.ContainerConditionHealthy,
		})
	}

	td.ContainerDefinitions = append(td.ContainerDefinitions, sc)

	return nil
}

// resolveTargets determines which containers should be patched. If names is
// empty, all existing containers are targets. Otherwise, only the specified
// names are used (and each must exist in the task definition).
func resolveTargets(td *TaskDefinition, names []string) (map[string]bool, error) {
	existing := make(map[string]bool, len(td.ContainerDefinitions))
	for _, c := range td.ContainerDefinitions {
		existing[aws.ToString(c.Name)] = true
	}

	targets := make(map[string]bool)
	if len(names) == 0 {
		for name := range existing {
			targets[name] = true
		}
		return targets, nil
	}

	for _, name := range names {
		if !existing[name] {
			return nil, fmt.Errorf("container %q not found in task definition", name)
		}
		targets[name] = true
	}
	return targets, nil
}

// wrapCommand prepends the ptrace-shim to the original container command.
// ECS Command is exec-form ([]string), so each element is passed directly
// as an argv entry with no shell interpretation.
func wrapCommand(cmd []string) []string {
	return append([]string{"/tmp/shared/ptrace-shim"}, cmd...)
}

// sidecarLogConfig returns a deep copy of the first target container's
// LogConfiguration, with the awslogs stream prefix overridden to sidecarName
// so sidecar streams are distinguishable from app streams. Returns nil if no
// target has a log configuration.
func sidecarLogConfig(td *TaskDefinition, targets map[string]bool) *ecstypes.LogConfiguration {
	for i := range td.ContainerDefinitions {
		if targets[aws.ToString(td.ContainerDefinitions[i].Name)] {
			src := td.ContainerDefinitions[i].LogConfiguration
			if src == nil {
				continue
			}
			copied := *src
			logOpts := make(map[string]string, len(src.Options))
			for k, v := range src.Options {
				logOpts[k] = v
			}
			if src.LogDriver == ecstypes.LogDriverAwslogs {
				logOpts["awslogs-stream-prefix"] = sidecarName
			}
			copied.Options = logOpts
			if len(src.SecretOptions) > 0 {
				copied.SecretOptions = make([]ecstypes.Secret, len(src.SecretOptions))
				copy(copied.SecretOptions, src.SecretOptions)
			}
			return &copied
		}
	}
	return nil
}

// addVolumeIfMissing adds a host volume with the given name if it does not
// already exist in the task definition.
func addVolumeIfMissing(td *TaskDefinition, name string) {
	for _, v := range td.Volumes {
		if aws.ToString(v.Name) == name {
			return
		}
	}
	td.Volumes = append(td.Volumes, ecstypes.Volume{
		Name: aws.String(name),
		Host: &ecstypes.HostVolumeProperties{},
	})
}
