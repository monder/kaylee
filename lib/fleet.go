package lib

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	fleetClient "github.com/coreos/fleet/client"
	fleetSchema "github.com/coreos/fleet/schema"
	"regexp"
	"strings"
	"time"
)

type Fleet struct {
	API    fleetClient.API
	Prefix string
}

func (fleet *Fleet) ScheduleUnit(unit *Unit, force bool) {
	specData, _ := json.Marshal(unit)
	specHash := sha1.Sum(specData)

	replicaUnit := true
	if unit.Replicas == 0 {
		unit.Replicas = 1
		replicaUnit = false
	}
	if unit.MaxReplicasPerHost == 0 {
		unit.MaxReplicasPerHost = unit.Replicas
	}
	// Make a list of units we should replace
	var unitsToRemove []string
	existingUnits, _ := fleet.API.Units()
	for _, u := range existingUnits {
		if strings.HasPrefix(u.Name, fmt.Sprintf("%s:%s:", fleet.Prefix, unit.Name)) {
			if !force && strings.HasPrefix(u.Name, fmt.Sprintf("%s:%s:%x:", fleet.Prefix, unit.Name, specHash[:3])) {
				// If the unit is already somewhere in the cluster
				return
			}
			unitsToRemove = append(unitsToRemove, u.Name)
		}
	}

	// Generate unique ids based on replica count and max replicas
	conflictIds := make([]string, unit.MaxReplicasPerHost)
	for i := 0; i < unit.MaxReplicasPerHost; i++ {
		r := make([]byte, 3)
		rand.Read(r) //TODO err
		conflictIds[i] = fmt.Sprintf("%x", r)
	}

	// Schedule replicas
	for i := 1; i <= unit.Replicas; i++ {
		unitName := fmt.Sprintf("%s:%s:%x:%s:%d.service",
			fleet.Prefix, unit.Name, specHash[:3], conflictIds[i%len(conflictIds)], i)
		var conflictStrings []string
		if replicaUnit {
			conflictStrings = append(conflictStrings, fmt.Sprintf("%s:%s:%x:%s:*.service",
				fleet.Prefix, unit.Name, specHash[:3], conflictIds[i%len(conflictIds)]))
		} else {
			conflictStrings = append(conflictStrings, fmt.Sprintf("%s:%s:*.service", fleet.Prefix, unit.Name))
		}
		for _, c := range unit.Conflicts {
			conflictStrings = append(conflictStrings, fmt.Sprintf("%s:%s:*.service", fleet.Prefix, c))
		}
		fleetUnit := makeFleetUnit(unitName, unit, conflictStrings)
		err := fleet.API.CreateUnit(fleetUnit)
		if err != nil {
			fmt.Println("Unable to create unit:", err)
		}
		fleet.waitForUnitStart(unitName)
		if len(unitsToRemove) > 0 {
			fmt.Println("Deleting unit:", unitsToRemove[0])
			fleet.API.DestroyUnit(unitsToRemove[0])
			unitsToRemove = unitsToRemove[1:]
		} else {
			fmt.Println("No more units to remove")
		}
	}
	for _, unit := range unitsToRemove {
		fmt.Println("Deleting unit:", unit)
		fleet.API.DestroyUnit(unit)
	}
}

func (fleet *Fleet) waitForUnitStart(name string) {
	fmt.Println("Waiting for unit to start:", name)
	prevState := "undefined"
	for i := 0; i < 60; i++ {
		currentState := "unknown"
		states, err := fleet.API.UnitStates()
		if err != nil {
			fmt.Println("Unable to retrieve unit state")
			continue
		}
		for _, state := range states {
			if state.Name == name {
				currentState = state.SystemdSubState
				break
			}
		}
		if currentState != prevState {
			if prevState != "undefined" {
				fmt.Print("\n")
			}
			fmt.Printf("%s: ", currentState)
			prevState = currentState
		} else {
			fmt.Print(".")
		}
		if currentState == "running" {
			fmt.Print("\n")
			fmt.Println("Unit started:", name)
			return
		}
		time.Sleep(time.Second)
	}
	fmt.Println("Unable to schedule unit:", name)
}

func makeFleetUnit(name string, spec *Unit, conflictStrings []string) *fleetSchema.Unit {
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

	for _, c := range conflictStrings {
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
