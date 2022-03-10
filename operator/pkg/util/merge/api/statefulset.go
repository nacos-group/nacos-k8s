package api

import (
	v1 "k8s.io/api/apps/v1"
	"encoding/json"
)

// StatefulSetConfiguration holds the optional custom StatefulSet
// that should be merged into the operator created one.
type StatefulSetConfiguration struct {
	// +kubebuilder:pruning:PreserveUnknownFields
	SpecWrapper StatefulSetSpecWrapper `json:"spec"`
}

type StatefulSetSpecWrapper struct {
	Spec v1.StatefulSetSpec `json:"-"`
}

// MarshalJSON defers JSON encoding to the wrapped map
func (m *StatefulSetSpecWrapper) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Spec)
}

// UnmarshalJSON will decode the data into the wrapped map
func (m *StatefulSetSpecWrapper) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &m.Spec)
}

func (m *StatefulSetSpecWrapper) DeepCopy() *StatefulSetSpecWrapper {
	return &StatefulSetSpecWrapper{
		Spec: m.Spec,
	}
}