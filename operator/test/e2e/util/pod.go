package util

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// 等待指定数量的pod，全部都是ready状态
func WaitforPodsRunning(clientSet *kubernetes.Clientset, namespace string, opts metav1.ListOptions, size int, timout time.Duration) (*v1.PodList, error) {
	timeStart := time.Now()
	fmt.Printf("begin to wait %v\n", timeStart)
	for {
		podList, err := clientSet.CoreV1().Pods(namespace).List(context.TODO(), opts)
		if err != nil {
			return nil, err
		}
		curRunNum := 0
		if len(podList.Items) == size {
			for _, pod := range podList.Items {
				if pod.Status.Phase == "Running" {
					curRunNum++
				}
			}
			if curRunNum == size {
				return podList, nil
			}
		}
		if timeStart.Add(timout).Before(time.Now()) {
			fmt.Printf("timeout %v\n", time.Now())
			return nil, fmt.Errorf("timeout")
		}
		fmt.Printf("wait for pod to ready %v\n", time.Now())
		time.Sleep(time.Second)
	}
}
