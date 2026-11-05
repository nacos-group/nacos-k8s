package k8s

import (
	"bytes"
	"encoding/json"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func replicasOrDefault(replicas *int32) int32 {
	if replicas == nil {
		return 1
	}
	return *replicas
}

// Compare container configuration, not the whole PodSpec: the API server fills
// unrelated Pod defaults. Normalize on copies so reconciliation never changes
// either the stored object or the desired object. JSON also equates omitted and
// empty optional lists/maps while preserving container and command ordering.
func containersEqual(old, desired []corev1.Container) bool {
	if len(old) != len(desired) {
		return false
	}
	for i := range old {
		a, _ := json.Marshal(containerWithDefaults(old[i]))
		b, _ := json.Marshal(containerWithDefaults(desired[i]))
		if !bytes.Equal(a, b) {
			return false
		}
	}
	return true
}

// Defaults follow the Kubernetes version used by client-go:
// https://github.com/kubernetes/kubernetes/blob/v1.21.4/pkg/apis/core/v1/defaults.go
func containerWithDefaults(container corev1.Container) *corev1.Container {
	c := container.DeepCopy()
	if c.ImagePullPolicy == "" {
		image := strings.SplitN(c.Image, "@", 2)
		tag := ""
		if colon := strings.LastIndex(image[0], ":"); colon > strings.LastIndex(image[0], "/") {
			tag = image[0][colon+1:]
		}
		c.ImagePullPolicy = corev1.PullIfNotPresent
		if tag == "latest" || (tag == "" && len(image) == 1) {
			c.ImagePullPolicy = corev1.PullAlways
		}
	}
	if c.TerminationMessagePath == "" {
		c.TerminationMessagePath = corev1.TerminationMessagePathDefault
	}
	if c.TerminationMessagePolicy == "" {
		c.TerminationMessagePolicy = corev1.TerminationMessageReadFile
	}
	for i := range c.Ports {
		if c.Ports[i].Protocol == "" {
			c.Ports[i].Protocol = corev1.ProtocolTCP
		}
	}
	for i := range c.Env {
		if source := c.Env[i].ValueFrom; source != nil {
			if source.FieldRef != nil && source.FieldRef.APIVersion == "" {
				source.FieldRef.APIVersion = "v1"
			}
			if source.ResourceFieldRef != nil && source.ResourceFieldRef.Divisor.IsZero() {
				source.ResourceFieldRef.Divisor = resource.MustParse("1")
			}
		}
	}
	for _, list := range []corev1.ResourceList{c.Resources.Limits, c.Resources.Requests} {
		for name, quantity := range list {
			quantity.RoundUp(resource.Milli)
			list[name] = quantity
		}
	}
	for _, probe := range []*corev1.Probe{c.LivenessProbe, c.ReadinessProbe, c.StartupProbe} {
		if probe == nil {
			continue
		}
		if probe.TimeoutSeconds == 0 {
			probe.TimeoutSeconds = 1
		}
		if probe.PeriodSeconds == 0 {
			probe.PeriodSeconds = 10
		}
		if probe.SuccessThreshold == 0 {
			probe.SuccessThreshold = 1
		}
		if probe.FailureThreshold == 0 {
			probe.FailureThreshold = 3
		}
		defaultHTTPGet(probe.HTTPGet)
	}
	if c.Lifecycle != nil {
		for _, hook := range []*corev1.Handler{c.Lifecycle.PostStart, c.Lifecycle.PreStop} {
			if hook != nil {
				defaultHTTPGet(hook.HTTPGet)
			}
		}
	}
	return c
}

func defaultHTTPGet(action *corev1.HTTPGetAction) {
	if action == nil {
		return
	}
	if action.Path == "" {
		action.Path = "/"
	}
	if action.Scheme == "" {
		action.Scheme = corev1.URISchemeHTTP
	}
}
