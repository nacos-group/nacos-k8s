package merge

import (
	"sort"

	"nacos.io/nacos-operator/pkg/util/contains"
	corev1 "k8s.io/api/core/v1"
)

func PodSpec(original, override corev1.PodSpec) corev1.PodSpec {
	merged := original

	merged.Volumes = Volumes(original.Volumes, override.Volumes)
	merged.Containers = Containers(original.Containers, override.Containers)
	merged.InitContainers = Containers(original.InitContainers, override.InitContainers)

	if override.EphemeralContainers != nil {
		merged.EphemeralContainers = EphemeralContainers(original.EphemeralContainers, override.EphemeralContainers)
	}

	if override.RestartPolicy != "" {
		merged.RestartPolicy = override.RestartPolicy
	}

	if override.TerminationGracePeriodSeconds != nil {
		merged.TerminationGracePeriodSeconds = override.TerminationGracePeriodSeconds
	}
	if override.ActiveDeadlineSeconds != nil {
		merged.ActiveDeadlineSeconds = override.ActiveDeadlineSeconds
	}

	if override.DNSPolicy != "" {
		merged.DNSPolicy = override.DNSPolicy
	}

	if override.NodeSelector != nil {
		merged.NodeSelector = StringToStringMap(original.NodeSelector, override.NodeSelector)
	}

	if override.ServiceAccountName != "" {
		merged.ServiceAccountName = override.ServiceAccountName
	}

	if override.DeprecatedServiceAccount != "" {
		merged.DeprecatedServiceAccount = override.DeprecatedServiceAccount
	}

	if override.AutomountServiceAccountToken != nil {
		merged.AutomountServiceAccountToken = override.AutomountServiceAccountToken
	}

	if override.NodeName != "" {
		merged.NodeName = override.NodeName
	}

	if override.HostNetwork {
		merged.HostNetwork = override.HostNetwork
	}

	if override.HostPID {
		merged.HostPID = override.HostPID
	}

	if override.ShareProcessNamespace != nil {
		merged.ShareProcessNamespace = override.ShareProcessNamespace
	}

	if override.SecurityContext != nil {
		merged.SecurityContext = override.SecurityContext
	}

	if override.ImagePullSecrets != nil {
		merged.ImagePullSecrets = override.ImagePullSecrets
	}

	if override.Hostname != "" {
		merged.Hostname = override.Hostname
	}

	if override.Subdomain != "" {
		merged.Subdomain = override.Subdomain
	}

	if override.Affinity != nil {
		merged.Affinity = Affinity(original.Affinity, override.Affinity)
	}

	if override.SchedulerName != "" {
		merged.SchedulerName = override.SchedulerName
	}

	merged.Tolerations = Tolerations(original.Tolerations, override.Tolerations)

	merged.HostAliases = HostAliases(original.HostAliases, override.HostAliases)

	if override.PriorityClassName != "" {
		merged.PriorityClassName = override.PriorityClassName
	}

	if override.Priority != nil {
		merged.Priority = override.Priority
	}

	if override.DNSConfig != nil {
		merged.DNSConfig = PodDNSConfig(original.DNSConfig, override.DNSConfig)
	}

	if override.ReadinessGates != nil {
		merged.ReadinessGates = override.ReadinessGates
	}

	if override.RuntimeClassName != nil {
		merged.RuntimeClassName = override.RuntimeClassName
	}

	if override.EnableServiceLinks != nil {
		merged.EnableServiceLinks = override.EnableServiceLinks
	}

	if override.PreemptionPolicy != nil {
		merged.PreemptionPolicy = override.PreemptionPolicy
	}

	if override.Overhead != nil {
		merged.Overhead = override.Overhead
	}

	if override.TopologySpreadConstraints != nil {
		merged.TopologySpreadConstraints = TopologySpreadConstraints(original.TopologySpreadConstraints, override.TopologySpreadConstraints)
	}

	return merged
}
func PodTemplateSpecs(original, override corev1.PodTemplateSpec) corev1.PodTemplateSpec {
	merged := original

	merged.Annotations = StringToStringMap(original.Annotations, override.Annotations)
	merged.Labels = StringToStringMap(original.Labels, override.Labels)
	merged.Spec = PodSpec(merged.Spec,override.Spec)
	return merged
}

func TopologySpreadConstraints(original, override []corev1.TopologySpreadConstraint) []corev1.TopologySpreadConstraint {
	originalMap := createTopologySpreadConstraintMap(original)
	overrideMap := createTopologySpreadConstraintMap(override)

	mergedMap := map[string]corev1.TopologySpreadConstraint{}

	for k, v := range originalMap {
		mergedMap[k] = v
	}
	for k, v := range overrideMap {
		if originalValue, ok := mergedMap[k]; ok {
			mergedMap[k] = TopologySpreadConstraint(originalValue, v)
		} else {
			mergedMap[k] = v
		}
	}
	var mergedElements []corev1.TopologySpreadConstraint
	for _, v := range mergedMap {
		mergedElements = append(mergedElements, v)
	}
	return mergedElements
}

func TopologySpreadConstraint(original, override corev1.TopologySpreadConstraint) corev1.TopologySpreadConstraint {
	merged := original
	if override.LabelSelector != nil {
		merged.LabelSelector = override.LabelSelector
	}
	if override.MaxSkew != 0 {
		merged.MaxSkew = override.MaxSkew
	}
	if override.WhenUnsatisfiable != "" {
		merged.WhenUnsatisfiable = override.WhenUnsatisfiable
	}
	return merged
}

func createTopologySpreadConstraintMap(constraints []corev1.TopologySpreadConstraint) map[string]corev1.TopologySpreadConstraint {
	m := make(map[string]corev1.TopologySpreadConstraint)
	for _, v := range constraints {
		m[v.TopologyKey] = v
	}
	return m
}

// HostAliases merges two slices of HostAliases together. Any shared hostnames with a given
// ip are merged together into fewer entries.
func HostAliases(originalAliases, overrideAliases []corev1.HostAlias) []corev1.HostAlias {
	m := make(map[string]corev1.HostAlias)
	for _, original := range originalAliases {
		m[original.IP] = original
	}

	for _, override := range overrideAliases {
		if _, ok := m[override.IP]; ok {
			var mergedHostNames []string
			mergedHostNames = append(mergedHostNames, m[override.IP].Hostnames...)
			for _, hn := range override.Hostnames {
				if !contains.String(mergedHostNames, hn) {
					mergedHostNames = append(mergedHostNames, hn)
				}
			}
			m[override.IP] = corev1.HostAlias{
				IP:        override.IP,
				Hostnames: mergedHostNames,
			}
		} else {
			m[override.IP] = override
		}
	}

	var mergedHostAliases []corev1.HostAlias
	for _, v := range m {
		mergedHostAliases = append(mergedHostAliases, v)
	}

	sort.SliceStable(mergedHostAliases, func(i, j int) bool {
		return mergedHostAliases[i].IP < mergedHostAliases[j].IP
	})

	return mergedHostAliases
}

func PodDNSConfig(originalDNSConfig, overrideDNSConfig *corev1.PodDNSConfig) *corev1.PodDNSConfig {
	if overrideDNSConfig == nil {
		return originalDNSConfig
	}

	if originalDNSConfig == nil {
		return overrideDNSConfig
	}

	merged := originalDNSConfig.DeepCopy()

	if overrideDNSConfig.Options != nil {
		merged.Options = overrideDNSConfig.Options
	}

	if overrideDNSConfig.Nameservers != nil {
		merged.Nameservers = StringSlices(merged.Nameservers, overrideDNSConfig.Nameservers)
	}

	if overrideDNSConfig.Searches != nil {
		merged.Searches = StringSlices(merged.Searches, overrideDNSConfig.Searches)
	}

	return merged
}
