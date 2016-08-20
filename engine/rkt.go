package engine

import (
	"fmt"
	"regexp"
	"strings"

	fleetSchema "github.com/coreos/fleet/schema"
	"github.com/monder/kaylee/spec"
)

type RktEngine struct{}

func (*RktEngine) IsValidSpecType(s *spec.Spec) bool {
	return s.Engine == "" || s.Engine == "rkt"
}

func (*RktEngine) ValidateSpec(s *spec.Spec) error {
	if len(s.Apps) == 0 {
		return fmt.Errorf("There should be at least one app")
	}
	return nil
}

func (*RktEngine) GetFleetUnit(spec *spec.Spec, name string, conflicts []string) *fleetSchema.Unit {
	uuidFileName := regexp.MustCompile("[^a-zA-Z0-9_.-]").ReplaceAllLiteralString(name, "_")
	uuidFileName = regexp.MustCompile("\\.service$").ReplaceAllLiteralString(uuidFileName, "")
	uuidFile := "/var/run/kaylee_" + uuidFileName

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
		Section: "Service", Name: "Environment", Value: fmt.Sprintf("KAYLEE_ID=%s", uuidFileName),
	})

	options = append(options, &fleetSchema.UnitOption{
		Section: "Service", Name: "TimeoutStartSec", Value: "0",
	})

	for _, volume := range spec.Volumes {
		options = append(options, &fleetSchema.UnitOption{
			Section: "Service",
			Name:    "ExecStartPre",
			Value:   fmt.Sprintf("/var/lib/kaylee/plugins/volumes/%s %s %s", volume.Driver, volume.ID, volume.Options),
		})
		options = append(options, &fleetSchema.UnitOption{
			Section: "Service",
			Name:    "ExecStopPost",
			Value:   fmt.Sprintf("/var/lib/kaylee/plugins/volumes/%s -u %s", volume.Driver, volume.ID),
		})

		args = append(args, fmt.Sprintf("--volume %s,kind=host,source=/mnt/%s/%s", volume.ID, volume.Driver, volume.ID))
		args = append(args, fmt.Sprintf("--mount volume=%s,target=%s", volume.ID, volume.Path))
	}

	options = append(options, &fleetSchema.UnitOption{
		Section: "Service", Name: "ExecStartPre", Value: fmt.Sprintf("-/usr/bin/rkt stop --force=true --uuid-file=%s", uuidFile),
	})

	if spec.Net != "" {
		args = append(args, fmt.Sprintf("--net=%s", spec.Net))
	}
	args = append(args, "--insecure-options=image")
	args = append(args, "--inherit-env")
	args = append(args, fmt.Sprintf("--uuid-file-save=%s", uuidFile))

	for _, app := range spec.Apps {
		options = append(options, &fleetSchema.UnitOption{
			Section: "Service", Name: "ExecStartPre", Value: fmt.Sprintf("/usr/bin/rkt fetch --insecure-options=image %s", app.Image),
		})
		args = append(args, fmt.Sprintf("%s -- %s ---", app.Image, strings.Join(app.Args, " ")))
	}

	options = append(options, &fleetSchema.UnitOption{
		Section: "Service",
		Name:    "ExecStart",
		Value:   fmt.Sprintf("/usr/bin/rkt run %s", strings.Join(args, " ")),
	})

	options = append(options, &fleetSchema.UnitOption{
		Section: "Service", Name: "ExecStop", Value: fmt.Sprintf("-/usr/bin/rkt stop --uuid-file=%s", uuidFile),
	})
	options = append(options, &fleetSchema.UnitOption{
		Section: "Service", Name: "ExecStop", Value: fmt.Sprintf("-/usr/bin/rkt rm --uuid-file=%s", uuidFile),
	})
	options = append(options, &fleetSchema.UnitOption{
		Section: "Service", Name: "ExecStop", Value: fmt.Sprintf("-/usr/bin/rm %s", uuidFile),
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
