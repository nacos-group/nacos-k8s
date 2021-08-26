package util

import (
	"context"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func CreateNS(clientSet *kubernetes.Clientset, name string) error {
	ns := &v1.Namespace{}
	ns.Name = name
	_, err := clientSet.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
	return err
}

func CreateNSIfNotExist(clientSet *kubernetes.Clientset, name string) error {
	_, err := clientSet.CoreV1().Namespaces().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return CreateNS(clientSet, name)
		}
		return err
	}
	return err
}

func DeleteNS(clientSet *kubernetes.Clientset, name string) error {
	return clientSet.CoreV1().Namespaces().Delete(context.TODO(), name, metav1.DeleteOptions{})
}
