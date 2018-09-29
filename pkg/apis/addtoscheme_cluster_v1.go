/*
Copyright 2018 Anevia.
*/

package apis

import (
	"github.com/feloy/operator/pkg/apis/cluster/v1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, v1.SchemeBuilder.AddToScheme)
}
