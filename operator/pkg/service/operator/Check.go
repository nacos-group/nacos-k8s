package operator

import (
	"strings"

	corev1 "k8s.io/api/core/v1"

	log "github.com/go-logr/logr"
	nacosgroupv1alpha1 "nacos.io/nacos-operator/api/v1alpha1"
	myErrors "nacos.io/nacos-operator/pkg/errors"
	"nacos.io/nacos-operator/pkg/service/k8s"
	nacosClient "nacos.io/nacos-operator/pkg/service/nacos"
)

type ICheckClient interface {
	Check(nacos *nacosgroupv1alpha1.Nacos)
}

type CheckClient struct {
	k8sService  k8s.Services
	logger      log.Logger
	nacosClient nacosClient.NacosClient
}

func NewCheckClient(logger log.Logger, k8sService k8s.Services) *CheckClient {
	return &CheckClient{
		k8sService: k8sService,
		logger:     logger,
	}
}

func (c *CheckClient) CheckKind(nacos *nacosgroupv1alpha1.Nacos) []corev1.Pod {
	// 保证ss数量和cr副本数匹配
	ss, err := c.k8sService.GetStatefulSet(nacos.Namespace, nacos.Name)
	myErrors.EnsureNormal(err)

	if *ss.Spec.Replicas != *nacos.Spec.Replicas {
		panic(myErrors.New(myErrors.CODE_ERR_UNKNOW, "cr replicas is not equal ss replicas"))

	}

	// 检查正常的pod数量，根据实际情况。如果单实例，必须要有1个;集群要1/2以上
	pods, err := c.k8sService.GetStatefulSetReadPod(nacos.Namespace, nacos.Name)
	if len(pods) < (int(*nacos.Spec.Replicas)+1)/2 {
		panic(myErrors.New(myErrors.CODE_ERR_UNKNOW, "The number of ready pods is too less"))
	} else if len(pods) != int(*nacos.Spec.Replicas) {
		c.logger.V(0).Info("pod num is not right")
	}
	return pods
}

func (c *CheckClient) CheckNacos(nacos *nacosgroupv1alpha1.Nacos, pods []corev1.Pod) {
	leader := ""
	nacos.Status.Conditions = []nacosgroupv1alpha1.Condition{}
	// 检查nacos是否访问通
	for _, pod := range pods {
		servers, err := c.nacosClient.GetClusterNodes(pod.Status.PodIP)
		myErrors.EnsureNormalMyError(err, myErrors.CODE_CLUSTER_FAILE)
		// 确保cr中实例个数和server数量相同
		myErrors.EnsureEqual(len(servers.Data), int(*nacos.Spec.Replicas), myErrors.CODE_CLUSTER_FAILE, "server num is not equal")
		for _, svc := range servers.Data {
			myErrors.EnsureEqual(svc.State, "UP", myErrors.CODE_CLUSTER_FAILE, "node is not up")
			if leader != "" {
				// 确保每个节点leader相同
				myErrors.EnsureEqual(leader, svc.ExtendInfo.RaftMetaData.MetaDataMap.NamingPersistentService.Leader,
					myErrors.CODE_CLUSTER_FAILE, "leader not equal")
			} else {
				leader = svc.ExtendInfo.RaftMetaData.MetaDataMap.NamingPersistentService.Leader
			}
			nacos.Status.Version = svc.ExtendInfo.Version
		}

		condition := nacosgroupv1alpha1.Condition{
			Status:   "true",
			Instance: pod.Status.PodIP,
			PodName:  pod.Name,
			NodeName: pod.Spec.NodeName,
		}
		leaderSplit := []string{}
		if strings.Index(leader, ".") > 0 {
			leaderSplit = strings.Split(leader, ".")
		} else {
			leaderSplit = strings.Split(leader, ":")
		}
		if len(leaderSplit) > 0 {
			if leaderSplit[0] == pod.Name {
				condition.Type = "leader"
			} else {
				condition.Type = "follower"
			}
		}
		nacos.Status.Conditions = append(nacos.Status.Conditions, condition)
	}

}
