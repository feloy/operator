/*
Copyright 2018 Anevia.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CdnClusterSpec defines the desired state of CdnCluster
type CdnClusterSpec struct {
	// Role of the CDN cluster, can be 'balancer' or 'cache'
	Role string `json:"role"`
}

// CdnClusterStatus defines the observed state of CdnCluster
type CdnClusterStatus struct {
	// State of the CDN cluster
	State string `json:"state"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CdnCluster is the Schema for the cdnclusters API
// +k8s:openapi-gen=true
type CdnCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CdnClusterSpec   `json:"spec,omitempty"`
	Status CdnClusterStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CdnClusterList contains a list of CdnCluster
type CdnClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CdnCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CdnCluster{}, &CdnClusterList{})
}
