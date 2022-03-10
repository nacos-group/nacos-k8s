package api

import (
	"encoding/json"

	v1 "k8s.io/api/core/v1"
)

// StatefulSetConfiguration holds the optional custom StatefulSet
// that should be merged into the operator created one.
type PodConfiguration struct {
	// +kubebuilder:pruning:PreserveUnknownFields
	SpecWrapper DeploymentSpecWrapper `json:"spec"`
}

type PodSpecWrapper struct {
	Spec v1.PodSpec `json:"-"`
}

// MarshalJSON defers JSON encoding to the wrapped map
func (m *PodSpecWrapper) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Spec)
}

// UnmarshalJSON will decode the data into the wrapped map
func (m *PodSpecWrapper) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &m.Spec)
}

func (m *PodSpecWrapper) DeepCopy() *PodSpecWrapper {
	return &PodSpecWrapper{
		Spec: m.Spec,
	}
}
