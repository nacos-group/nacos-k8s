package operator

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	log "github.com/go-logr/logr"
	nacosgroupv1alpha1 "nacos.io/nacos-operator/api/v1alpha1"
	myErrors "nacos.io/nacos-operator/pkg/errors"
	"nacos.io/nacos-operator/pkg/service/k8s"
)

type IStatusClient interface {
}

type StatusClient struct {
	logger log.Logger
	client client.Client
}

func NewStatusClient(logger log.Logger, k8sService k8s.Services, client client.Client) *StatusClient {
	return &StatusClient{
		client: client,
		logger: logger,
	}
}

// 更新状态
func (c *StatusClient) UpdateStatusRunning(nacos *nacosgroupv1alpha1.Nacos) {
	c.updateLastEvent(nacos, 200, "", true)
	nacos.Status.Phase = nacosgroupv1alpha1.PhaseRunning
	// TODO
	myErrors.EnsureNormal(c.client.Status().Update(context.TODO(), nacos))
}

// 更新状态
func (c *StatusClient) UpdateStatus(nacos *nacosgroupv1alpha1.Nacos) {
	// TODO
	myErrors.EnsureNormal(c.client.Status().Update(context.TODO(), nacos))
}

func (c *StatusClient) UpdateExceptionStatus(nacos *nacosgroupv1alpha1.Nacos, err *myErrors.Err) {
	c.updateLastEvent(nacos, err.Code, err.Msg, false)
	// 设置为异常状态
	nacos.Status.Phase = nacosgroupv1alpha1.PhaseFailed
	e := c.client.Status().Update(context.TODO(), nacos)
	if e != nil {
		c.logger.V(-1).Info(e.Error())
	}

}

const EVENT_MAX_SIZE = 10

func (c *StatusClient) updateLastEvent(nacos *nacosgroupv1alpha1.Nacos, code int, msg string, status bool) {
	var event nacosgroupv1alpha1.Event
	if len(nacos.Status.Event) > EVENT_MAX_SIZE {
		nacos.Status.Event = append(nacos.Status.Event[:0], nacos.Status.Event[1:]...)

	}
	if len(nacos.Status.Event) == 0 {
		event = nacosgroupv1alpha1.Event{
			Code: code,
			FirstAppearTime: metav1.Time{
				Time: time.Now()},
			Message: msg,
		}
		nacos.Status.Event = append(nacos.Status.Event, event)
	} else {
		// 获取最近的event
		event = nacos.Status.Event[len(nacos.Status.Event)-1]
	}

	// 如果是已经存在的，就更新时间
	if event.Code == code {
		event.LastTransitionTime.Time = time.Now()
		event.Message = msg
		nacos.Status.Event[len(nacos.Status.Event)-1] = event
	} else {
		event = nacosgroupv1alpha1.Event{
			Code:    code,
			Status:  status,
			Message: msg,
			LastTransitionTime: metav1.Time{
				Time: time.Now(),
			},
			FirstAppearTime: metav1.Time{
				Time: time.Now(),
			},
		}
		nacos.Status.Event = append(nacos.Status.Event, event)
	}

}
