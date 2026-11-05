package k8s

import (
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func testStatefulSet() *appsv1.StatefulSet {
	var replicas int32 = 4
	return &appsv1.StatefulSet{Spec: appsv1.StatefulSetSpec{
		Replicas: &replicas,
		Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name: "nacos", Image: "nacos/nacos-server:v2.4.3",
				Command:       []string{"sh", "-c", "old startup"},
				Env:           []corev1.EnvVar{{Name: "MODE", Value: "cluster"}},
				LivenessProbe: &corev1.Probe{Handler: corev1.Handler{HTTPGet: &corev1.HTTPGetAction{Port: intstr.FromInt(8848)}}},
			}},
			InitContainers: []corev1.Container{{Name: "mysql-check", Image: "mysql:8.0", Command: []string{"mysqlcheck"}}},
		}},
	}}
}

func TestCheckStatefulSetChanges(t *testing.T) {
	tests := []struct {
		name   string
		change func(*appsv1.StatefulSet)
		want   operator
	}{
		{"unchanged", func(s *appsv1.StatefulSet) {}, None},
		{"command", func(s *appsv1.StatefulSet) { s.Spec.Template.Spec.Containers[0].Command[2] = "fixed startup" }, Update},
		{"image", func(s *appsv1.StatefulSet) { s.Spec.Template.Spec.Containers[0].Image = "nacos/nacos-server:v2.4.4" }, Update},
		{"args", func(s *appsv1.StatefulSet) { s.Spec.Template.Spec.Containers[0].Args = []string{"new"} }, Update},
		{"init image", func(s *appsv1.StatefulSet) { s.Spec.Template.Spec.InitContainers[0].Image = "mysql:8.1" }, Update},
		{"init command", func(s *appsv1.StatefulSet) { s.Spec.Template.Spec.InitContainers[0].Command = []string{"fixed-check"} }, Update},
		{"init env", func(s *appsv1.StatefulSet) {
			s.Spec.Template.Spec.InitContainers[0].Env = []corev1.EnvVar{{Name: "MYSQL_HOST", Value: "db"}}
		}, Update},
		{"init removed", func(s *appsv1.StatefulSet) { s.Spec.Template.Spec.InitContainers = nil }, Update},
		{"init added", func(s *appsv1.StatefulSet) {
			s.Spec.Template.Spec.InitContainers = append(s.Spec.Template.Spec.InitContainers, corev1.Container{Name: "extra", Image: "check:1"})
		}, Update},
		{"init volume mount", func(s *appsv1.StatefulSet) {
			s.Spec.Template.Spec.InitContainers[0].VolumeMounts = []corev1.VolumeMount{{Name: "db", MountPath: "/db"}}
		}, Update},
		{"liveness changed", func(s *appsv1.StatefulSet) { s.Spec.Template.Spec.Containers[0].LivenessProbe.HTTPGet.Path = "/health" }, Update},
		{"liveness removed", func(s *appsv1.StatefulSet) { s.Spec.Template.Spec.Containers[0].LivenessProbe = nil }, Update},
		{"readiness added", func(s *appsv1.StatefulSet) {
			s.Spec.Template.Spec.Containers[0].ReadinessProbe = &corev1.Probe{Handler: corev1.Handler{Exec: &corev1.ExecAction{Command: []string{"ready"}}}}
		}, Update},
		{"startup added", func(s *appsv1.StatefulSet) {
			s.Spec.Template.Spec.Containers[0].StartupProbe = &corev1.Probe{Handler: corev1.Handler{Exec: &corev1.ExecAction{Command: []string{"started"}}}}
		}, Update},
		{"probe timeout", func(s *appsv1.StatefulSet) { s.Spec.Template.Spec.Containers[0].LivenessProbe.TimeoutSeconds = 2 }, Update},
		{"replicas", func(s *appsv1.StatefulSet) { *s.Spec.Replicas = 3 }, Update},
		{"resources", func(s *appsv1.StatefulSet) {
			s.Spec.Template.Spec.Containers[0].Resources.Requests = corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("1Gi")}
		}, Update},
		{"env", func(s *appsv1.StatefulSet) { s.Spec.Template.Spec.Containers[0].Env[0].Value = "standalone" }, Update},
		{"PVC added", func(s *appsv1.StatefulSet) { s.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{{}} }, Delete},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := testStatefulSet()
			new := old.DeepCopy()
			tt.change(new)
			oldBefore, newBefore := old.DeepCopy(), new.DeepCopy()
			if got := checkSts(old, new); got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(old, oldBefore) || !reflect.DeepEqual(new, newBefore) {
				t.Fatal("change detection mutated its inputs")
			}
		})
	}
}

func TestCheckStatefulSetIgnoresAPIDefaults(t *testing.T) {
	desired := testStatefulSet()
	desired.Spec.Template.Spec.Containers[0].Env = append(desired.Spec.Template.Spec.Containers[0].Env,
		corev1.EnvVar{Name: "POD_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}}})
	desired.Spec.Template.Spec.Containers[0].Ports = []corev1.ContainerPort{{ContainerPort: 8848}}
	desired.Spec.Template.Spec.Containers[0].Args = []string{}
	live := desired.DeepCopy()
	for _, containers := range [][]corev1.Container{live.Spec.Template.Spec.Containers, live.Spec.Template.Spec.InitContainers} {
		for i := range containers {
			containers[i].ImagePullPolicy = corev1.PullIfNotPresent
			containers[i].TerminationMessagePath = "/dev/termination-log"
			containers[i].TerminationMessagePolicy = corev1.TerminationMessageReadFile
		}
	}
	container := &live.Spec.Template.Spec.Containers[0]
	container.Args = nil
	container.Env[1].ValueFrom.FieldRef.APIVersion = "v1"
	container.Ports[0].Protocol = corev1.ProtocolTCP
	probe := container.LivenessProbe
	probe.TimeoutSeconds, probe.PeriodSeconds, probe.SuccessThreshold, probe.FailureThreshold = 1, 10, 1, 3
	probe.HTTPGet.Path, probe.HTTPGet.Scheme = "/", corev1.URISchemeHTTP
	beforeLive, beforeDesired := live.DeepCopy(), desired.DeepCopy()
	if got := checkSts(live, desired); got != None {
		t.Fatalf("API defaults must not trigger an update: %v", got)
	}
	if !reflect.DeepEqual(live, beforeLive) || !reflect.DeepEqual(desired, beforeDesired) {
		t.Fatal("default normalization mutated its inputs")
	}
}

func TestCheckStatefulSetPVCChangeTakesPrecedence(t *testing.T) {
	storageClass := "standard"
	old := testStatefulSet()
	old.Spec.VolumeClaimTemplates = []corev1.PersistentVolumeClaim{{Spec: corev1.PersistentVolumeClaimSpec{
		StorageClassName: &storageClass,
		Resources:        corev1.ResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("1Gi")}},
	}}}
	if got := checkSts(old, old.DeepCopy()); got != None {
		t.Fatalf("unchanged PVC must not be recreated: %v", got)
	}
	for _, change := range []func(*appsv1.StatefulSet){
		func(s *appsv1.StatefulSet) {
			s.Spec.VolumeClaimTemplates[0].Spec.Resources.Requests[corev1.ResourceStorage] = resource.MustParse("2Gi")
		},
		func(s *appsv1.StatefulSet) { *s.Spec.VolumeClaimTemplates[0].Spec.StorageClassName = "fast" },
	} {
		new := old.DeepCopy()
		change(new)
		new.Spec.Template.Spec.Containers[0].Image = "nacos:new"
		if got := checkSts(old, new); got != Delete {
			t.Fatalf("PVC changes must retain delete/recreate precedence: %v", got)
		}
	}
}

func TestCheckStatefulSetImagePullDefaults(t *testing.T) {
	for _, tt := range []struct {
		image  string
		policy corev1.PullPolicy
	}{
		{"nacos/nacos-server:v2.4.3", corev1.PullIfNotPresent},
		{"nacos/nacos-server:latest", corev1.PullAlways},
		{"registry.example:5000/nacos", corev1.PullAlways},
		{"nacos", corev1.PullAlways},
		{"nacos@sha256:abcdef", corev1.PullIfNotPresent},
		{"nacos:latest@sha256:abcdef", corev1.PullAlways},
	} {
		t.Run(tt.image, func(t *testing.T) {
			desired := testStatefulSet()
			desired.Spec.Template.Spec.Containers[0].Image = tt.image
			desired.Spec.Template.Spec.InitContainers[0].Image = tt.image
			live := desired.DeepCopy()
			live.Spec.Template.Spec.Containers[0].ImagePullPolicy = tt.policy
			live.Spec.Template.Spec.InitContainers[0].ImagePullPolicy = tt.policy
			if got := checkSts(live, desired); got != None {
				t.Fatalf("default pull policy should not trigger an update: %v", got)
			}
		})
	}
}

func TestCheckStatefulSetAllContainersAndOrdering(t *testing.T) {
	old := testStatefulSet()
	old.Spec.Template.Spec.Containers = append(old.Spec.Template.Spec.Containers, corev1.Container{Name: "sidecar", Image: "sidecar:1"})
	old.Spec.Template.Spec.InitContainers = append(old.Spec.Template.Spec.InitContainers, corev1.Container{Name: "prepare", Image: "prepare:1"})
	for _, tt := range []struct {
		name   string
		change func(*appsv1.StatefulSet)
	}{
		{"second container", func(s *appsv1.StatefulSet) { s.Spec.Template.Spec.Containers[1].Image = "sidecar:2" }},
		{"removed container", func(s *appsv1.StatefulSet) { s.Spec.Template.Spec.Containers = s.Spec.Template.Spec.Containers[:1] }},
		{"init order", func(s *appsv1.StatefulSet) {
			c := s.Spec.Template.Spec.InitContainers
			c[0], c[1] = c[1], c[0]
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			desired := old.DeepCopy()
			tt.change(desired)
			if got := checkSts(old, desired); got != Update {
				t.Fatalf("container change was missed: %v", got)
			}
		})
	}
}

func TestCheckStatefulSetNestedDefaults(t *testing.T) {
	desired := testStatefulSet()
	c := &desired.Spec.Template.Spec.InitContainers[0]
	c.Env = []corev1.EnvVar{{Name: "CPU_LIMIT", ValueFrom: &corev1.EnvVarSource{
		ResourceFieldRef: &corev1.ResourceFieldSelector{Resource: "limits.cpu"},
	}}}
	c.Lifecycle = &corev1.Lifecycle{PreStop: &corev1.Handler{HTTPGet: &corev1.HTTPGetAction{Port: intstr.FromInt(8848)}}}
	c.ReadinessProbe = desired.Spec.Template.Spec.Containers[0].LivenessProbe.DeepCopy()
	c.StartupProbe = c.ReadinessProbe.DeepCopy()
	c.Resources.Limits = corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("0.0001")}
	live := desired.DeepCopy()
	c = &live.Spec.Template.Spec.InitContainers[0]
	c.Env[0].ValueFrom.ResourceFieldRef.Divisor = resource.MustParse("1")
	c.Lifecycle.PreStop.HTTPGet.Path = "/"
	c.Lifecycle.PreStop.HTTPGet.Scheme = corev1.URISchemeHTTP
	c.Resources.Limits[corev1.ResourceCPU] = resource.MustParse("1m")
	for _, probe := range []*corev1.Probe{c.ReadinessProbe, c.StartupProbe} {
		probe.TimeoutSeconds, probe.PeriodSeconds, probe.SuccessThreshold, probe.FailureThreshold = 1, 10, 1, 3
		probe.HTTPGet.Path, probe.HTTPGet.Scheme = "/", corev1.URISchemeHTTP
	}
	beforeLive, beforeDesired := live.DeepCopy(), desired.DeepCopy()
	if got := checkSts(live, desired); got != None {
		t.Fatalf("nested API defaults should not trigger an update: %v", got)
	}
	if !reflect.DeepEqual(live, beforeLive) || !reflect.DeepEqual(desired, beforeDesired) {
		t.Fatal("nested normalization mutated its inputs")
	}
	// Requests inherited when creating a Pod must not be imposed on its template.
	live.Spec.Template.Spec.InitContainers[0].Resources.Requests = corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("1m")}
	if got := checkSts(live, desired); got != Update {
		t.Fatalf("an explicit resource request change must still be detected: %v", got)
	}
}

func TestCheckStatefulSetDefaultReplicasAndEmptyInitContainers(t *testing.T) {
	desired := testStatefulSet()
	desired.Spec.Replicas = nil
	desired.Spec.Template.Spec.InitContainers = nil
	live := desired.DeepCopy()
	var one int32 = 1
	live.Spec.Replicas = &one
	live.Spec.Template.Spec.InitContainers = []corev1.Container{}
	if got := checkSts(live, desired); got != None {
		t.Fatalf("replica default or empty init list should not trigger an update: %v", got)
	}
}
