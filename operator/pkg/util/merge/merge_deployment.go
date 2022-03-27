package merge

import appsv1 "k8s.io/api/apps/v1"

// StatefulSetSpecs merges two StatefulSetSpecs together.
func DeploymentSpecs(defaultSpec, overrideSpec appsv1.DeploymentSpec) appsv1.DeploymentSpec {
	mergedSpec := defaultSpec
	if overrideSpec.Replicas != nil {
		mergedSpec.Replicas = overrideSpec.Replicas
	}

	mergedSpec.Selector = LabelSelectors(defaultSpec.Selector, overrideSpec.Selector)

	if overrideSpec.RevisionHistoryLimit != nil {
		mergedSpec.RevisionHistoryLimit = overrideSpec.RevisionHistoryLimit
	}

	mergedSpec.Template = PodTemplateSpecs(defaultSpec.Template, overrideSpec.Template)
	return mergedSpec
}
