package k8s

import (
	log "github.com/go-logr/logr"
	"k8s.io/client-go/kubernetes"
)

// Service is the K8s service entrypoint.
type Services interface {
	ConfigMap
	StatefulSet
	Service
	Job
}

type services struct {
	ConfigMap
	StatefulSet
	Service
	Job
}

// New returns a new Kubernetes service.
func NewK8sService(kubecli kubernetes.Interface, logger log.Logger) Services {
	return &services{
		ConfigMap:   NewConfigMapService(kubecli, logger),
		StatefulSet: NewStatefulSetService(kubecli, logger),
		Service:     NewServiceService(kubecli, logger),
		Job:         NewJobService(kubecli, logger),
	}
}
