package engine

import (
	"fmt"
	"regexp"
	"strings"

	fleetSchema "github.com/coreos/fleet/schema"
	"github.com/monder/kaylee/spec"
)

type DockerEngine struct{}

func (*DockerEngine) IsValidSpecType(s *spec.Spec) bool {
	return s.Engine == "docker"
}

func (*DockerEngine) ValidateSpec(s *spec.Spec) error {
	if len(s.Apps) == 0 {
		return fmt.Errorf("There should be at least one app")
	}
	if len(s.Apps) > 1 {
		return fmt.Errorf("Docker endigne does not support more than one app")
	}
	return nil
}

func (*DockerEngine) GetFleetUnit(spec *spec.Spec, name string, conflicts []string) *fleetSchema.Unit {
	dockerName := regexp.MustCompile("[^a-zA-Z0-9_.-]").ReplaceAllLiteralString(name, "_")
	dockerName = regexp.MustCompile("\\.service$").ReplaceAllLiteralString(dockerName, "")

	var args []string

	for _, arg := range spec.Args {
		args = append(args, arg)
	}

	var options []*fleetSchema.UnitOption
	options = append(options, &fleetSchema.UnitOption{
		Section: "Unit", Name: "Requires", Value: "flanneld.service",
	})
	options = append(options, &fleetSchema.UnitOption{
		Section: "Unit", Name: "After", Value: "flanneld.service",
	})

	for _, env := range spec.EnvFiles {
		options = append(options, &fleetSchema.UnitOption{
			Section: "Service", Name: "EnvironmentFile", Value: env,
		})
	}
	for _, env := range spec.Env {
		options = append(options, &fleetSchema.UnitOption{
			Section: "Service", Name: "Environment", Value: fmt.Sprintf("%s=%s", env.Name, env.Value),
		})
	}

	options = append(options, &fleetSchema.UnitOption{
		Section: "Service", Name: "TimeoutStartSec", Value: "0",
	})

	for _, volume := range spec.Volumes {
		options = append(options, &fleetSchema.UnitOption{
			Section: "Service",
			Name:    "ExecStartPre",
			Value:   fmt.Sprintf("/usr/bin/docker volume create --name=%s --driver=%s --opt=%s", volume.ID, volume.Driver, volume.Options),
		})
		args = append(args, fmt.Sprintf("-v %s:%s", volume.ID, volume.Path))
	}

	options = append(options, &fleetSchema.UnitOption{
		Section: "Service", Name: "ExecStartPre", Value: fmt.Sprintf("/usr/bin/docker pull %s", spec.Apps[0].Image),
	})
	options = append(options, &fleetSchema.UnitOption{
		Section: "Service", Name: "ExecStartPre", Value: fmt.Sprintf("-/usr/bin/docker kill %s", dockerName),
	})
	options = append(options, &fleetSchema.UnitOption{
		Section: "Service", Name: "ExecStartPre", Value: fmt.Sprintf("-/usr/bin/docker rm %s", dockerName),
	})

	options = append(options, &fleetSchema.UnitOption{
		Section: "Service",
		Name:    "ExecStart",
		Value:   fmt.Sprintf("/usr/bin/docker run %s --rm --name %s %s %s", strings.Join(args, " "), dockerName, spec.Apps[0].Image, spec.Apps[0].Args),
	})

	options = append(options, &fleetSchema.UnitOption{
		Section: "Service", Name: "ExecStop", Value: fmt.Sprintf("-/usr/bin/docker stop %s", dockerName),
	})
	options = append(options, &fleetSchema.UnitOption{
		Section: "Service", Name: "Restart", Value: "always",
	})
	options = append(options, &fleetSchema.UnitOption{
		Section: "Service", Name: "RestartSec", Value: "30",
	})

	for _, machine := range spec.Machine {
		options = append(options, &fleetSchema.UnitOption{
			Section: "X-Fleet", Name: "MachineMetadata", Value: machine,
		})
	}
	if spec.MachineID != "" {
		options = append(options, &fleetSchema.UnitOption{
			Section: "X-Fleet", Name: "MachineID", Value: spec.MachineID,
		})
	}
	if spec.Global {
		options = append(options, &fleetSchema.UnitOption{
			Section: "X-Fleet", Name: "Global", Value: "true",
		})
	}

	for _, c := range conflicts {
		options = append(options, &fleetSchema.UnitOption{
			Section: "X-Fleet", Name: "Conflicts", Value: c,
		})
	}
	return &fleetSchema.Unit{
		DesiredState: "launched",
		Options:      options,
		Name:         name,
	}
}
