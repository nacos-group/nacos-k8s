package api

// StatefulSetConfiguration holds the optional custom StatefulSet
// that should be merged into the operator created one.
type PodConfiguration struct {
	// +kubebuilder:pruning:PreserveUnknownFields
	SpecWrapper DeploymentSpecWrapper `json:"spec"`
}
