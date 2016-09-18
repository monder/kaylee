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

	var args []string

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
			Section: "Service", Name: "Environment", Value: fmt.Sprintf("%q", fmt.Sprintf("%s=%s", env.Name, env.Value)),
		})
	}
	options = append(options, &fleetSchema.UnitOption{
		Section: "Service", Name: "Environment", Value: fmt.Sprintf("KAYLEE_ID=%s", uuidFileName),
	})

	options = append(options, &fleetSchema.UnitOption{
		Section: "Service", Name: "TimeoutStartSec", Value: "0",
	})

	options = append(options, &fleetSchema.UnitOption{
		Section: "Service", Name: "KillMode", Value: "mixed",
	})

	for volumeIndex, volume := range spec.Volumes {
		if volume.Driver == "" {
			// Assume "host"
			args = append(args, fmt.Sprintf("--volume kaylee-volume-%d,kind=host,source=%q", volumeIndex, volume.Source))
			args = append(args, fmt.Sprintf("--mount volume=kaylee-volume-%d,target=%q", volumeIndex, volume.Path))
		} else {
			if strings.HasPrefix(volume.Driver, "kaylee-mount-") {
				options = append(options, &fleetSchema.UnitOption{
					// TODO BindsTo=?
					Section: "Unit", Name: "Requires", Value: fmt.Sprintf("%s@%s.service", volume.Driver, volume.ID),
				})
				options = append(options, &fleetSchema.UnitOption{
					Section: "Unit", Name: "After", Value: fmt.Sprintf("%s@%s.service", volume.Driver, volume.ID),
				})
			} else {
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
			}

			args = append(args, fmt.Sprintf("--volume kaylee-volume-%d,kind=host,source=/mnt/%s/%s/%s", volumeIndex, volume.Driver, volume.ID, volume.Source))
			args = append(args, fmt.Sprintf("--mount volume=kaylee-volume-%d,target=%s", volumeIndex, volume.Path))
		}
	}

	if spec.Net != "" {
		args = append(args, fmt.Sprintf("--net=%s", spec.Net))
	}
	args = append(args, "--insecure-options=image,ondisk")
	args = append(args, "--inherit-env")

	for _, arg := range spec.Args {
		args = append(args, arg)
	}

	for _, app := range spec.Apps {
		options = append(options, &fleetSchema.UnitOption{
			Section: "Service", Name: "ExecStartPre", Value: fmt.Sprintf("/usr/bin/rkt fetch --insecure-options=image,ondisk %s", app.Image),
		})
		imageOptions := ""
		if app.Cmd != "" {
			imageOptions = fmt.Sprintf("--exec=%q", app.Cmd)
		}
		args = append(args, fmt.Sprintf("%s %s -- %s ---", app.Image, imageOptions, strings.Join(app.Args, " ")))
	}

	options = append(options, &fleetSchema.UnitOption{
		Section: "Service",
		Name:    "ExecStart",
		Value:   fmt.Sprintf("/usr/bin/rkt run %s", strings.Join(args, " ")),
	})

	options = append(options, &fleetSchema.UnitOption{
		Section: "Service", Name: "ExecStopPost", Value: "/usr/bin/rkt gc",
	})
	options = append(options, &fleetSchema.UnitOption{
		Section: "Service", Name: "ExecStopPost", Value: "/usr/bin/rkt image gc",
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
