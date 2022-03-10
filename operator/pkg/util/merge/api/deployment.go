package api

import (
	"encoding/json"

	v1 "k8s.io/api/apps/v1"
)

// StatefulSetConfiguration holds the optional custom StatefulSet
// that should be merged into the operator created one.
type DeploymentConfiguration struct {
	// +kubebuilder:pruning:PreserveUnknownFields
	SpecWrapper DeploymentSpecWrapper `json:"spec"`
}

type DeploymentSpecWrapper struct {
	Spec v1.DeploymentSpec `json:"-"`
}

// MarshalJSON defers JSON encoding to the wrapped map
func (m *DeploymentSpecWrapper) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Spec)
}

// UnmarshalJSON will decode the data into the wrapped map
func (m *DeploymentSpecWrapper) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &m.Spec)
}

func (m *DeploymentSpecWrapper) DeepCopy() *DeploymentSpecWrapper {
	return &DeploymentSpecWrapper{
		Spec: m.Spec,
	}
}
