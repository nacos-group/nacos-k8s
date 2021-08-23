package k8s

import (
	"context"
	log "github.com/go-logr/logr"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

type Job interface {
	GetJob(namespace string, name string) (*batchv1.Job, error)
	CreateJob(namespace string, job *batchv1.Job) error
	CreateIfNotExistsJob(namespace string, job *batchv1.Job) error
}

type JobService struct {
	kubeClient kubernetes.Interface
	logger     log.Logger
}

func NewJobService(kubeClient kubernetes.Interface, logger log.Logger) *JobService {
	logger = logger.WithValues("service", "k8s.job")
	return &JobService{
		kubeClient: kubeClient,
		logger:     logger,
	}
}

func (s *JobService) GetJob(namespace string, name string) (*batchv1.Job, error) {
	job, err := s.kubeClient.BatchV1().Jobs(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return job, err
}

func (s *JobService) CreateJob(namespace string, job *batchv1.Job) error {
	_, err := s.kubeClient.BatchV1().Jobs(namespace).Create(context.TODO(), job, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	klog.V(2).Infof("create job,namespace: %s  name: %s", namespace, job.Name)
	return nil
}

func (s *JobService) CreateIfNotExistsJob(namespace string, job *batchv1.Job) error {
	if _, err := s.GetJob(namespace, job.Name); err != nil {
		// If no resource we need to create.
		if errors.IsNotFound(err) {
			return s.CreateJob(namespace, job)
		}
		return err
	}
	return nil
}
