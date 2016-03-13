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

func (fleet *Fleet) ScheduleUnit(unit *Unit) {
	replicaUnit := true
	if unit.Spec.Replicas == 0 {
		unit.Spec.Replicas = 1
		replicaUnit = false
	}
	if unit.Spec.MaxReplicasPerHost == 0 {
		unit.Spec.MaxReplicasPerHost = unit.Spec.Replicas
	}
	// Make a list of units we should replace
	var unitsToRemove []string
	existingUnits, _ := fleet.API.Units()
	for _, u := range existingUnits {
		if strings.HasPrefix(u.Name, fmt.Sprintf("%s:%s:", fleet.Prefix, unit.Name)) {
			unitsToRemove = append(unitsToRemove, u.Name)
		}
	}

	// Generate unique ids based on replica count and max replicas
	conflictIds := make([]string, unit.Spec.MaxReplicasPerHost)
	for i := 0; i < unit.Spec.MaxReplicasPerHost; i++ {
		r := make([]byte, 3)
		rand.Read(r) //TODO err
		conflictIds[i] = fmt.Sprintf("%x", r)
	}

	// Schedule replicas
	specData, _ := json.Marshal(unit)
	specHash := sha1.Sum(specData)
	for i := 1; i <= unit.Spec.Replicas; i++ {
		unitName := fmt.Sprintf("%s:%s:%x:%s:%d.service",
			fleet.Prefix, unit.Name, specHash[:3], conflictIds[i%len(conflictIds)], i)
		conflictString := fmt.Sprintf("%s:%s:%x:%s:*.service",
			fleet.Prefix, unit.Name, specHash[:3], conflictIds[i%len(conflictIds)])
		if !replicaUnit {
			conflictString = fmt.Sprintf("%s:%s:*.service", fleet.Prefix, unit.Name)
		}
		fleetUnit := makeFleetUnit(unitName, unit, conflictString)
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

func makeFleetUnit(name string, spec *Unit, conflictString string) *fleetSchema.Unit {
	dockerName := regexp.MustCompile("[^a-zA-Z0-9_.-]").ReplaceAllLiteralString(name, "_")
	dockerName = regexp.MustCompile("\\.service$").ReplaceAllLiteralString(dockerName, "")

	var dockerArgs []string

	dockerArgs = append(dockerArgs, fmt.Sprintf("-l s7r.name=%s", spec.Name))

	for _, env := range spec.Spec.Env {
		dockerArgs = append(dockerArgs, fmt.Sprintf("-e %s=%s", env.Name, env.Value))
	}
	for _, label := range spec.Spec.Labels {
		dockerArgs = append(dockerArgs, fmt.Sprintf("-l %s=%s", label.Name, label.Value))
	}
	for _, arg := range spec.Spec.DockerArgs {
		dockerArgs = append(dockerArgs, arg)
	}

	var options []*fleetSchema.UnitOption
	options = append(options, &fleetSchema.UnitOption{
		Section: "Unit", Name: "Requires", Value: "flanneld.service",
	})
	options = append(options, &fleetSchema.UnitOption{
		Section: "Unit", Name: "After", Value: "flanneld.service",
	})
	options = append(options, &fleetSchema.UnitOption{
		Section: "Service", Name: "EnvironmentFile", Value: "/etc/environment",
	})
	options = append(options, &fleetSchema.UnitOption{
		Section: "Service", Name: "ExecStartPre", Value: fmt.Sprintf("/usr/bin/docker pull %s", spec.Spec.Image),
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
		Value:   fmt.Sprintf("/usr/bin/docker run --rm %s --name %s %s", strings.Join(dockerArgs, " "), dockerName, spec.Spec.Image),
	})

	options = append(options, &fleetSchema.UnitOption{
		Section: "Service", Name: "ExecStop", Value: fmt.Sprintf("-/usr/bin/docker stop %s", dockerName),
	})
	options = append(options, &fleetSchema.UnitOption{
		Section: "Service", Name: "Restart", Value: "always",
	})
	options = append(options, &fleetSchema.UnitOption{
		Section: "Service", Name: "RestartSec", Value: "10",
	})

	for _, machine := range spec.Spec.Machine {
		options = append(options, &fleetSchema.UnitOption{
			Section: "X-Fleet", Name: "MachineMetadata", Value: machine,
		})
	}
	if spec.Spec.MachineID != "" {
		options = append(options, &fleetSchema.UnitOption{
			Section: "X-Fleet", Name: "MachineID", Value: spec.Spec.MachineID,
		})
	}

	options = append(options, &fleetSchema.UnitOption{
		Section: "X-Fleet", Name: "Conflicts", Value: conflictString,
	})
	return &fleetSchema.Unit{
		DesiredState: "launched",
		Options:      options,
		Name:         name,
	}
}
