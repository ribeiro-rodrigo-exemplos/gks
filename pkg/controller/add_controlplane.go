package controller

import (
	"gitlab.globoi.com/tks/gks/gks-operator/pkg/controller/controlplane"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, controlplane.Add)
}
