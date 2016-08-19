package lib

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	fleetClient "github.com/coreos/fleet/client"
	"github.com/monder/kaylee/engine"
	"github.com/monder/kaylee/spec"
	"strings"
	"time"
)

type Fleet struct {
	API    fleetClient.API
	Prefix string
}

func (fleet *Fleet) ScheduleUnit(unit *spec.Spec, force bool) {
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
		fleetUnit, err := engine.GetFleetUnit(unit, unitName, conflictStrings)
		if err != nil {
			fmt.Println("Unable to create unit:", err)
		}
		err = fleet.API.CreateUnit(fleetUnit)
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
