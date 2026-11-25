package patcher

import (
	"context"
	"errors"
	"fmt"
	"slices"

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
	shimPath          = sharedMountPath + "/ptrace-shim"
	profilesMountPath = "/profiles"
)

// Patch modifies the given TaskDefinition in place, injecting the ARMO ptrace
// sidecar and volume-fixer init container. Target containers are wrapped to
// launch through the ptrace shim. On error the task definition is left
// unchanged.
func Patch(ctx context.Context, td *TaskDefinition, opts PatchOptions, sidecar SidecarConfig) error {
	for _, c := range td.ContainerDefinitions {
		if aws.ToString(c.Name) == sidecarName {
			return fmt.Errorf("task definition is already patched: container %q exists", sidecarName)
		}
	}

	targets, err := resolveTargets(td, opts.Containers)
	if err != nil {
		return err
	}

	// Work out every target's new argv before touching td, so a failed
	// lookup leaves the task definition as it was.
	var arch ecstypes.CPUArchitecture
	if td.RuntimePlatform != nil {
		arch = td.RuntimePlatform.CpuArchitecture
	}
	wrapped := make(map[string]argv, len(targets))
	var errs []error
	for i := range td.ContainerDefinitions {
		c := &td.ContainerDefinitions[i]
		if !targets[aws.ToString(c.Name)] {
			continue
		}
		a, err := wrapContainer(ctx, c, arch, opts.Resolver)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		wrapped[aws.ToString(c.Name)] = a
	}
	if err := errors.Join(errs...); err != nil {
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
			Command:     []string{"CMD", "/usr/bin/ptrace-agent", "--health", "--shim"},
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
			Command:   []string{"sh", "-c", fmt.Sprintf("chmod -R 777 %s && chown -R 1000:1000 %s", sharedMountPath, sharedMountPath)},
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

		c.EntryPoint = wrapped[name].entryPoint
		c.Command = wrapped[name].command

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

// argv is a container's patched entryPoint and command.
type argv struct {
	entryPoint []string
	command    []string
}

// wrapContainer returns the entryPoint/command that start c through the
// ptrace shim with the argv it had before patching. ECS runs
// entryPoint+command; without an entryPoint it runs the image ENTRYPOINT
// followed by command or, if that is empty too, the image CMD. Setting
// entryPoint makes Docker ignore the image CMD, so it is copied into
// command explicitly.
func wrapContainer(ctx context.Context, c *ecstypes.ContainerDefinition, arch ecstypes.CPUArchitecture, resolver ImageConfigResolver) (argv, error) {
	if len(c.EntryPoint) > 0 {
		if c.EntryPoint[0] == shimPath {
			return argv{entryPoint: c.EntryPoint, command: c.Command}, nil
		}
		return argv{entryPoint: append([]string{shimPath}, c.EntryPoint...), command: c.Command}, nil
	}

	image := aws.ToString(c.Image)
	fail := func(cause error) (argv, error) {
		return argv{}, fmt.Errorf("container %q: cannot read ENTRYPOINT/CMD of image %q: %v. "+
			"Set \"entryPoint\" and \"command\" explicitly for this container in the task definition, "+
			"or leave it out of the targets with --container", aws.ToString(c.Name), image, cause)
	}
	if resolver == nil {
		return fail(errors.New("no image resolver configured"))
	}
	cfg, err := resolver.Resolve(ctx, image, arch)
	if err != nil {
		return fail(err)
	}

	command := c.Command
	if len(command) == 0 {
		command = slices.Clone(cfg.Cmd)
	}
	if len(cfg.Entrypoint) == 0 && len(command) == 0 {
		return fail(errors.New("image defines no ENTRYPOINT or CMD"))
	}
	return argv{entryPoint: append([]string{shimPath}, cfg.Entrypoint...), command: command}, nil
}

// sidecarLogConfig returns a deep copy of the LogConfiguration of the first
// target container that has one set, with the awslogs stream prefix
// overridden to sidecarName so sidecar streams are distinguishable from app
// streams. Returns nil if no target has a log configuration.
func sidecarLogConfig(td *TaskDefinition, targets map[string]bool) *ecstypes.LogConfiguration {
	for i := range td.ContainerDefinitions {
		if targets[aws.ToString(td.ContainerDefinitions[i].Name)] {
			src := td.ContainerDefinitions[i].LogConfiguration
			if src == nil {
				continue
			}
			copied := *src
			// Only allocate an Options map when there is actually
			// something to put in it. This preserves a nil source
			// Options on non-awslogs drivers so the rendered task
			// definition does not gain a spurious "options: {}".
			if len(src.Options) > 0 || src.LogDriver == ecstypes.LogDriverAwslogs {
				logOpts := make(map[string]string, len(src.Options)+1)
				for k, v := range src.Options {
					logOpts[k] = v
				}
				if src.LogDriver == ecstypes.LogDriverAwslogs {
					logOpts["awslogs-stream-prefix"] = sidecarName
				}
				copied.Options = logOpts
			}
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
