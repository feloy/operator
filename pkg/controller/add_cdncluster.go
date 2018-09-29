/*
Copyright 2018 Anevia.
*/

package controller

import (
	"github.com/feloy/operator/pkg/controller/cdncluster"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, cdncluster.Add)
}
