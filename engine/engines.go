package engine

import (
	"fmt"

	fleetSchema "github.com/coreos/fleet/schema"
	"github.com/monder/kaylee/spec"
)

type Engine interface {
	IsValidSpecType(s *spec.Spec) bool
	ValidateSpec(s *spec.Spec) error
	GetFleetUnit(s *spec.Spec, name string, conflicts []string) *fleetSchema.Unit
}

var engines []Engine

func init() {
	engines = []Engine{
		&RktEngine{},
	}
}

func GetFleetUnit(s *spec.Spec, name string, conflicts []string) (*fleetSchema.Unit, error) {
	for _, e := range engines {
		if e.IsValidSpecType(s) {
			err := e.ValidateSpec(s)
			if err != nil {
				return nil, err
			}
			return e.GetFleetUnit(s, name, conflicts), nil
		}
	}
	return nil, fmt.Errorf("Invalid engine %s\n", s.Engine)
}
