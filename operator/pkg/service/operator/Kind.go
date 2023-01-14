package operator

import (
	"fmt"
	"io"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"nacos.io/nacos-operator/pkg/util/merge"
	"path/filepath"
	"strconv"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	myErrors "nacos.io/nacos-operator/pkg/errors"

	log "github.com/go-logr/logr"
	appv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	nacosgroupv1alpha1 "nacos.io/nacos-operator/api/v1alpha1"
	"nacos.io/nacos-operator/pkg/service/k8s"
)

const TYPE_STAND_ALONE = "standalone"
const TYPE_CLUSTER = "cluster"
const NACOS = "nacos"
const NACOS_PORT = 8848
const RAFT_PORT = 7848
const NEW_RAFT_PORT = 9848

// 导入的sql文件名称
const SQL_FILE_NAME = "nacos-mysql.sql"

var initScrit = `array=(%s)
succ = 0

for element in ${array[@]} 
do
  while true
  do
    ping $element -c 1 > /dev/stdout
    if [[ $? -eq 0 ]]; then
      echo $element "all domain ready"
      break
    else
      echo $element "wait for other domain ready"
    fi
    sleep 1
  done
done
sleep 1

echo "init success"`

type IKindClient interface {
	Ensure(nacos nacosgroupv1alpha1.Nacos)
	EnsureStatefulset(nacos nacosgroupv1alpha1.Nacos)
	EnsureConfigmap(nacos nacosgroupv1alpha1.Nacos)
}

type KindClient struct {
	k8sService k8s.Services
	logger     log.Logger
	scheme     *runtime.Scheme
}

func NewKindClient(logger log.Logger, k8sService k8s.Services, scheme *runtime.Scheme) *KindClient {
	return &KindClient{
		k8sService: k8sService,
		logger:     logger,
		scheme:     scheme,
	}
}

func (e *KindClient) generateLabels(name string, component string) map[string]string {
	return map[string]string{
		"app":        name,
		"middleware": NACOS,
		"component":  component,
	}
}

func (e *KindClient) generateAnnoation() map[string]string {
	return map[string]string{}
}

// 合并cr中的label 和 固定的label
func (e *KindClient) MergeLabels(allLabels ...map[string]string) map[string]string {
	res := map[string]string{}
	for _, labels := range allLabels {
		if labels != nil {
			for k, v := range labels {
				res[k] = v
			}
		}
	}
	return res
}

func (e *KindClient) generateName(nacos *nacosgroupv1alpha1.Nacos) string {
	return nacos.Name
}

func (e *KindClient) generateHeadlessSvcName(nacos *nacosgroupv1alpha1.Nacos) string {
	return fmt.Sprintf("%s-headless", nacos.Name)
}
func (e *KindClient) generateClientSvcName(nacos *nacosgroupv1alpha1.Nacos) string {
	return fmt.Sprintf("%s-client", nacos.Name)
}

// CR格式验证
func (e *KindClient) ValidationField(nacos *nacosgroupv1alpha1.Nacos) {

	setDefaultValue := []func(nacos *nacosgroupv1alpha1.Nacos){
		setDefaultNacosType,
		setDefaultMysql,
		setDefaultCertification,
	}

	for _, f := range setDefaultValue {
		f(nacos)
	}
}

func setDefaultNacosType(nacos *nacosgroupv1alpha1.Nacos) {
	// 默认设置单节点
	if nacos.Spec.Type == "" {
		nacos.Spec.Type = "standalone"
	}
}

func setDefaultCertification(nacos *nacosgroupv1alpha1.Nacos) {
	// 默认设置认证参数
	if nacos.Spec.Certification.Enabled {
		if nacos.Spec.Certification.Token == "" {
			nacos.Spec.Certification.Token = "SecretKey012345678901234567890123456789012345678901234567890123456789"
		}
		if nacos.Spec.Certification.TokenExpireSeconds == "" {
			nacos.Spec.Certification.TokenExpireSeconds = "18000"
		}
	}
}

func setDefaultMysql(nacos *nacosgroupv1alpha1.Nacos) {
	// 默认设置内置数据库
	if nacos.Spec.Database.TypeDatabase == "" {
		nacos.Spec.Database.TypeDatabase = "embedded"
	}
	// mysql设置默认值
	if nacos.Spec.Database.TypeDatabase == "mysql" {
		if nacos.Spec.Database.MysqlHost == "" {
			nacos.Spec.Database.MysqlHost = "127.0.0.1"
		}
		if nacos.Spec.Database.MysqlUser == "" {
			nacos.Spec.Database.MysqlUser = "root"
		}
		if nacos.Spec.Database.MysqlDb == "" {
			nacos.Spec.Database.MysqlDb = "nacos"
		}
		if nacos.Spec.Database.MysqlPassword == "" {
			nacos.Spec.Database.MysqlPassword = "123456"
		}
		if nacos.Spec.Database.MysqlPort == "" {
			nacos.Spec.Database.MysqlPort = "3306"
		}
	}
}

func (e *KindClient) EnsureStatefulsetCluster(nacos *nacosgroupv1alpha1.Nacos) {
	ss := e.buildStatefulset(nacos)
	ss = e.buildStatefulsetCluster(nacos, ss)
	ss.Spec.Template.Spec = merge.PodSpec(ss.Spec.Template.Spec, nacos.Spec.K8sWrapper.PodSpec.Spec)
	myErrors.EnsureNormal(e.k8sService.CreateOrUpdateStatefulSet(nacos.Namespace, ss))
}

func (e *KindClient) EnsureStatefulset(nacos *nacosgroupv1alpha1.Nacos) {
	ss := e.buildStatefulset(nacos)
	ss.Spec.Template.Spec = merge.PodSpec(ss.Spec.Template.Spec, nacos.Spec.K8sWrapper.PodSpec.Spec)
	myErrors.EnsureNormal(e.k8sService.CreateOrUpdateStatefulSet(nacos.Namespace, ss))
}

func (e *KindClient) EnsureService(nacos *nacosgroupv1alpha1.Nacos) {
	ss := e.buildService(nacos)
	myErrors.EnsureNormal(e.k8sService.CreateIfNotExistsService(nacos.Namespace, ss))
}

func (e *KindClient) EnsureServiceCluster(nacos *nacosgroupv1alpha1.Nacos) {
	ss := e.buildService(nacos)
	myErrors.EnsureNormal(e.k8sService.CreateOrUpdateService(nacos.Namespace, ss))
}

func (e *KindClient) EnsureClientService(nacos *nacosgroupv1alpha1.Nacos) {
	ss := e.buildClientService(nacos)
	myErrors.EnsureNormal(e.k8sService.CreateIfNotExistsService(nacos.Namespace, ss))
}

func (e *KindClient) EnsureHeadlessServiceCluster(nacos *nacosgroupv1alpha1.Nacos) {
	ss := e.buildService(nacos)
	ss = e.buildHeadlessServiceCluster(ss, nacos)
	myErrors.EnsureNormal(e.k8sService.CreateOrUpdateService(nacos.Namespace, ss))
}

func (e *KindClient) EnsureConfigmap(nacos *nacosgroupv1alpha1.Nacos) {
	if nacos.Spec.Config != "" {
		cm := e.buildConfigMap(nacos)
		myErrors.EnsureNormal(e.k8sService.CreateIfNotExistsConfigMap(nacos.Namespace, cm))
	}
}

func (e *KindClient) EnsureMysqlConfigMap(nacos *nacosgroupv1alpha1.Nacos) {
	cm := e.buildMysqlConfigMap(nacos)
	myErrors.EnsureNormal(e.k8sService.CreateIfNotExistsConfigMap(nacos.Namespace, cm))
}

func (e *KindClient) EnsureJob(nacos *nacosgroupv1alpha1.Nacos) {
	// 使用job执行SQL脚本的逻辑
	job := e.buildJob(nacos)
	job.Spec.Template.Spec = merge.PodSpec(job.Spec.Template.Spec, nacos.Spec.K8sWrapper.PodSpec.Spec)
	myErrors.EnsureNormal(e.k8sService.CreateIfNotExistsJob(nacos.Namespace, job))
}

// buildSqlConfigMap 创建用于保存待导入的sql的configmap
func (e *KindClient) buildMysqlConfigMap(nacos *nacosgroupv1alpha1.Nacos) *v1.ConfigMap {
	labels := e.generateLabels(nacos.Name, NACOS)
	labels = e.MergeLabels(nacos.Labels, labels)

	// 创建ConfigMap用于保存sql语句
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nacos.Name + "-mysql-sql-init",
			Namespace: nacos.Namespace,
			Labels:    labels,
		},

		Data: map[string]string{
			"SQL_SCRIPT": readSql(SQL_FILE_NAME),
		},
	}
	myErrors.EnsureNormal(controllerutil.SetControllerReference(nacos, cm, e.scheme))
	return cm
}

func (e *KindClient) buildJob(nacos *nacosgroupv1alpha1.Nacos) *batchv1.Job {
	labels := e.generateLabels(nacos.Name, NACOS)
	labels = e.MergeLabels(nacos.Labels, labels)

	// 创建Job用于向数据库中导入sql
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nacos.Name + "-mysql-sql-init",
			Namespace: nacos.Namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: nacos.Namespace,
				},
				Spec: v1.PodSpec{
					InitContainers: []v1.Container{
						{
							Name:  "mysql-ping-database",
							Image: nacos.Spec.MysqlInitImage,
							Env: []v1.EnvVar{
								{
									Name:  "MYSQL_HOST",
									Value: nacos.Spec.Database.MysqlHost,
								},
								{
									Name:  "MYSQL_DB",
									Value: nacos.Spec.Database.MysqlDb,
								},
								{
									Name:  "MYSQL_PORT",
									Value: nacos.Spec.Database.MysqlPort,
								},
								{
									Name:  "MYSQL_USER",
									Value: nacos.Spec.Database.MysqlUser,
								},
								{
									Name:  "MYSQL_PASS",
									Value: nacos.Spec.Database.MysqlPassword,
								},
							},
							Command: []string{
								"/bin/sh",
								"-c",
								"while ! mysqladmin ping --host=\"${MYSQL_HOST}\" --port=\"${MYSQL_PORT}\" --user=\"${MYSQL_USER}\" --password=\"${MYSQL_PASS}\" ; do echo \"check mysql\"; sleep 1; done",
							},
						},
						{
							Name:  "mysql-create-database",
							Image: nacos.Spec.MysqlInitImage,
							Env: []v1.EnvVar{
								{
									Name:  "MYSQL_HOST",
									Value: nacos.Spec.Database.MysqlHost,
								},
								{
									Name:  "MYSQL_DB",
									Value: nacos.Spec.Database.MysqlDb,
								},
								{
									Name:  "MYSQL_PORT",
									Value: nacos.Spec.Database.MysqlPort,
								},
								{
									Name:  "MYSQL_USER",
									Value: nacos.Spec.Database.MysqlUser,
								},
								{
									Name:  "MYSQL_PASS",
									Value: nacos.Spec.Database.MysqlPassword,
								},
							},
							// 判断数据库是否存在，不存在则创建
							Command: []string{
								"/bin/sh",
								"-c",
								"until mysql -u\"${MYSQL_USER}\" -p\"${MYSQL_PASS}\" -h\"${MYSQL_HOST}\" -P\"${MYSQL_PORT}\" -e\"create database if not exists \"${MYSQL_DB}\"\"; do echo waiting for database creation...; sleep 2; done;",
							},
						},
					},
					Containers: []v1.Container{
						{
							Name:  "mysql-sql-init",
							Image: nacos.Spec.MysqlInitImage,
							Env: []v1.EnvVar{
								{
									Name:  "MYSQL_HOST",
									Value: nacos.Spec.Database.MysqlHost,
								},
								{
									Name:  "MYSQL_DB",
									Value: nacos.Spec.Database.MysqlDb,
								},
								{
									Name:  "MYSQL_PORT",
									Value: nacos.Spec.Database.MysqlPort,
								},
								{
									Name:  "MYSQL_USER",
									Value: nacos.Spec.Database.MysqlUser,
								},
								{
									Name:  "MYSQL_PASS",
									Value: nacos.Spec.Database.MysqlPassword,
								},
								{
									Name: "SQL_SCRIPT",
									ValueFrom: &v1.EnvVarSource{
										ConfigMapKeyRef: &v1.ConfigMapKeySelector{
											LocalObjectReference: v1.LocalObjectReference{
												Name: nacos.Name + "-mysql-sql-init",
											},
											Key: "SQL_SCRIPT",
										},
									},
									//Value: readSql(SQL_FILE_NAME),
								},
							},
							// 导入nacos-mysql.sql中的数据
							Command: []string{
								"/bin/sh",
								"-c",
								"mysql -u\"${MYSQL_USER}\" -p\"${MYSQL_PASS}\" -h\"${MYSQL_HOST}\" -P\"${MYSQL_PORT}\" -D\"${MYSQL_DB}\" -e\"${SQL_SCRIPT}\";",
							},
						},
					},
					RestartPolicy: "Never",
				},
			},
		},
	}

	myErrors.EnsureNormal(controllerutil.SetControllerReference(nacos, job, e.scheme))
	return job
}

func readSql(sqlFileName string) string {
	// abspath：项目的根路径
	abspath, _ := filepath.Abs("")
	bytes, err := os.ReadFile(abspath + "/config/sql/" + sqlFileName)
	if err != nil {
		fmt.Printf("read sql file failed, err: %s", err.Error())
		return ""
	}

	return string(bytes)
}

func (e *KindClient) buildService(nacos *nacosgroupv1alpha1.Nacos) *v1.Service {
	labels := e.generateLabels(nacos.Name, NACOS)
	labels = e.MergeLabels(nacos.Labels, labels)

	annotations := e.MergeLabels(e.generateAnnoation(), nacos.Annotations)

	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        nacos.Name,
			Namespace:   nacos.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: v1.ServiceSpec{
			PublishNotReadyAddresses: true,
			Ports: []v1.ServicePort{
				{
					Name:     "client",
					Port:     NACOS_PORT,
					Protocol: "TCP",
				},
				{
					Name:     "rpc",
					Port:     RAFT_PORT,
					Protocol: "TCP",
				},
				{
					Name:     "new-rpc",
					Port:     NEW_RAFT_PORT,
					Protocol: "TCP",
				},
			},
			Selector: labels,
		},
	}
	myErrors.EnsureNormal(controllerutil.SetControllerReference(nacos, svc, e.scheme))
	return svc
}

func (e *KindClient) buildClientService(nacos *nacosgroupv1alpha1.Nacos) *v1.Service {
	labels := e.generateLabels(nacos.Name, NACOS)
	labels = e.MergeLabels(nacos.Labels, labels)

	annotations := e.MergeLabels(e.generateAnnoation(), nacos.Annotations)

	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        e.generateClientSvcName(nacos),
			Namespace:   nacos.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: v1.ServiceSpec{
			PublishNotReadyAddresses: true,
			Ports: []v1.ServicePort{
				{
					Name:     "client",
					Port:     NACOS_PORT,
					Protocol: "TCP",
				},
			},
			Selector: labels,
		},
	}
	//client-service提供双栈
	var ipf = make([]v1.IPFamily, 0)
	ipf = append(ipf, v1.IPv4Protocol)
	ipf = append(ipf, v1.IPv6Protocol)
	svc.Spec.IPFamilies = ipf
	var ipPli = v1.IPFamilyPolicyPreferDualStack
	svc.Spec.IPFamilyPolicy = &ipPli
	myErrors.EnsureNormal(controllerutil.SetControllerReference(nacos, svc, e.scheme))
	return svc
}

func (e *KindClient) buildStatefulset(nacos *nacosgroupv1alpha1.Nacos) *appv1.StatefulSet {
	// 生成label
	labels := e.generateLabels(nacos.Name, NACOS)
	// 合并cr中原有的label
	labels = e.MergeLabels(nacos.Labels, labels)

	// 设置默认的环境变量
	env := append(nacos.Spec.Env, v1.EnvVar{
		Name:  "PREFER_HOST_MODE",
		Value: "hostname",
	})

	// 设置认证环境变量
	if nacos.Spec.Certification.Enabled {
		env = append(env, v1.EnvVar{
			Name:  "NACOS_AUTH_ENABLE",
			Value: strconv.FormatBool(nacos.Spec.Certification.Enabled),
		})

		env = append(env, v1.EnvVar{
			Name:  "NACOS_AUTH_TOKEN_EXPIRE_SECONDS",
			Value: nacos.Spec.Certification.TokenExpireSeconds,
		})

		env = append(env, v1.EnvVar{
			Name:  "NACOS_AUTH_TOKEN",
			Value: nacos.Spec.Certification.Token,
		})

		env = append(env, v1.EnvVar{
			Name:  "NACOS_AUTH_CACHE_ENABLE",
			Value: strconv.FormatBool(nacos.Spec.Certification.CacheEnabled),
		})
	}

	// 数据库设置
	if nacos.Spec.Database.TypeDatabase == "embedded" {
		env = append(env, v1.EnvVar{
			Name:  "EMBEDDED_STORAGE",
			Value: "embedded",
		})
	} else if nacos.Spec.Database.TypeDatabase == "mysql" {

		env = append(env, v1.EnvVar{
			Name:  "SPRING_DATASOURCE_PLATFORM",
			Value: nacos.Spec.Database.TypeDatabase,
		})

		env = append(env, v1.EnvVar{
			Name:  "MYSQL_SERVICE_HOST",
			Value: nacos.Spec.Database.MysqlHost,
		})

		env = append(env, v1.EnvVar{
			Name:  "MYSQL_SERVICE_PORT",
			Value: nacos.Spec.Database.MysqlPort,
		})

		env = append(env, v1.EnvVar{
			Name:  "MYSQL_SERVICE_DB_NAME",
			Value: nacos.Spec.Database.MysqlDb,
		})

		env = append(env, v1.EnvVar{
			Name:  "MYSQL_SERVICE_USER",
			Value: nacos.Spec.Database.MysqlUser,
		})

		env = append(env, v1.EnvVar{
			Name:  "MYSQL_SERVICE_PASSWORD",
			Value: nacos.Spec.Database.MysqlPassword,
		})
	}

	// 启动模式 ，默认cluster
	if nacos.Spec.Type == TYPE_STAND_ALONE {
		env = append(env, v1.EnvVar{
			Name:  "MODE",
			Value: "standalone",
		})
	} else {
		env = append(env, v1.EnvVar{
			Name:  "NACOS_REPLICAS",
			Value: strconv.Itoa(int(*nacos.Spec.Replicas)),
		})
	}

	var ss = &appv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        e.generateName(nacos),
			Namespace:   nacos.Namespace,
			Labels:      labels,
			Annotations: nacos.Annotations,
		},
		Spec: appv1.StatefulSetSpec{
			PodManagementPolicy: "Parallel",
			Replicas:            nacos.Spec.Replicas,
			Selector:            &metav1.LabelSelector{MatchLabels: labels},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: v1.PodSpec{
					Volumes:      []v1.Volume{},
					NodeSelector: nacos.Spec.NodeSelector,
					Tolerations:  nacos.Spec.Tolerations,
					Affinity:     nacos.Spec.Affinity,
					Containers: []v1.Container{
						{
							Name:  nacos.Name,
							Image: nacos.Spec.Image,
							Lifecycle: &v1.Lifecycle{
								PreStop: &v1.Handler{
									Exec: &v1.ExecAction{
										Command: []string{
											"/bin/sh",
											"-c",
											"rm -rf /home/nacos/data/protocol/raft",
										},
									},
								},
							},
							Ports: []v1.ContainerPort{
								{
									Name:          "client",
									ContainerPort: NACOS_PORT,
									Protocol:      "TCP",
								},
								{
									Name:          "rpc",
									ContainerPort: RAFT_PORT,
									Protocol:      "TCP",
								},
								{
									Name:          "new-rpc",
									ContainerPort: NEW_RAFT_PORT,
									Protocol:      "TCP",
								},
							},
							Env:            env,
							LivenessProbe:  nacos.Spec.LivenessProbe,
							ReadinessProbe: nacos.Spec.ReadinessProbe,
							VolumeMounts:   []v1.VolumeMount{},
							Resources:      nacos.Spec.Resources,
						},
					},
				},
			},
		},
	}

	// 设置存储
	if nacos.Spec.Volume.Enabled {
		ss.Spec.VolumeClaimTemplates = append(ss.Spec.VolumeClaimTemplates, v1.PersistentVolumeClaim{
			Spec: v1.PersistentVolumeClaimSpec{
				//VolumeName:       "db",
				StorageClassName: nacos.Spec.Volume.StorageClass,
				AccessModes:      []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
				Resources: v1.ResourceRequirements{
					Requests: nacos.Spec.Volume.Requests,
				},
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:   "db",
				Labels: labels,
			},
		})

		localVolum := v1.VolumeMount{
			Name:      "db",
			MountPath: "/home/nacos/data",
		}
		ss.Spec.Template.Spec.Containers[0].VolumeMounts = append(ss.Spec.Template.Spec.Containers[0].VolumeMounts, localVolum)
	}

	//probe := &v1.Probe{
	//	InitialDelaySeconds: 10,
	//	PeriodSeconds:       5,
	//	TimeoutSeconds:      4,
	//	FailureThreshold:    5,
	//	Handler: v1.Handler{
	//		HTTPGet: &v1.HTTPGetAction{
	//			Port: intstr.IntOrString{IntVal: NACOS_PORT},
	//			Path: "/nacos/actuator/health/",
	//		},
	//		//TCPSocket: &v1.TCPSocketAction{
	//		//	Port: intstr.IntOrString{IntVal: NACOS_PORT},
	//		//},
	//	},
	//}

	//if nacos.Spec.LivenessProbe == nil {
	//	ss.Spec.Template.Spec.Containers[0].LivenessProbe = probe
	//}
	//if nacos.Spec.ReadinessProbe == nil {
	//	ss.Spec.Template.Spec.Containers[0].ReadinessProbe = probe
	//}

	if nacos.Spec.Config != "" {
		ss.Spec.Template.Spec.Volumes = append(ss.Spec.Template.Spec.Volumes, v1.Volume{
			Name: "config",
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: v1.LocalObjectReference{Name: nacos.Name},
					Items: []v1.KeyToPath{
						{
							Key:  "custom.properties",
							Path: "custom.properties",
						},
					},
				},
			},
		})
		ss.Spec.Template.Spec.Containers[0].VolumeMounts = append(ss.Spec.Template.Spec.Containers[0].VolumeMounts, v1.VolumeMount{
			Name:      "config",
			MountPath: "/home/nacos/init.d/custom.properties",
			SubPath:   "custom.properties",
		})
	}
	myErrors.EnsureNormal(controllerutil.SetControllerReference(nacos, ss, e.scheme))

	if nacos.Spec.Database.TypeDatabase == "mysql" && nacos.Spec.MysqlInitImage != "" {
		ss = e.AddCheckDatabase(nacos, ss)
	}
	return ss
}

func (e *KindClient) AddCheckDatabase(nacos *nacosgroupv1alpha1.Nacos, sts *appv1.StatefulSet) *appv1.StatefulSet {
	container := v1.Container{

		Name:  "mysql-check-database",
		Image: nacos.Spec.MysqlInitImage,
		Env: []v1.EnvVar{
			{
				Name:  "MYSQL_HOST",
				Value: nacos.Spec.Database.MysqlHost,
			},
			{
				Name:  "MYSQL_DB",
				Value: nacos.Spec.Database.MysqlDb,
			},
			{
				Name:  "MYSQL_PORT",
				Value: nacos.Spec.Database.MysqlPort,
			},
			{
				Name:  "MYSQL_USER",
				Value: nacos.Spec.Database.MysqlUser,
			},
			{
				Name:  "MYSQL_PASS",
				Value: nacos.Spec.Database.MysqlPassword,
			},
		},
		Command: []string{
			"/bin/sh",
			"-c",
			"while ! mysqlcheck --host=\"${MYSQL_HOST}\" --port=\"${MYSQL_PORT}\" --user=\"${MYSQL_USER}\" --password=\"${MYSQL_PASS}\" --databases \"${MYSQL_DB}\" ; do sleep 1; done"},
	}
	sts.Spec.Template.Spec.InitContainers = append(sts.Spec.Template.Spec.InitContainers, container)
	return sts
}

func (e *KindClient) buildConfigMap(nacos *nacosgroupv1alpha1.Nacos) *v1.ConfigMap {
	labels := e.generateLabels(nacos.Name, NACOS)
	labels = e.MergeLabels(nacos.Labels, labels)
	data := make(map[string]string)

	data["custom.properties"] = nacos.Spec.Config

	cm := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        e.generateName(nacos),
			Namespace:   nacos.Namespace,
			Labels:      labels,
			Annotations: nacos.Annotations,
		},
		Data: data,
	}
	myErrors.EnsureNormal(controllerutil.SetControllerReference(nacos, &cm, e.scheme))
	return &cm
}

func (e *KindClient) buildDefaultConfigMap(nacos *nacosgroupv1alpha1.Nacos) *v1.ConfigMap {
	labels := e.generateLabels(nacos.Name, NACOS)
	labels = e.MergeLabels(nacos.Labels, labels)
	data := make(map[string]string)

	// https://github.com/nacos-group/nacos-docker/blob/master/build/conf/application.properties
	data["application.properties"] = `# spring
	server.servlet.contextPath=${SERVER_SERVLET_CONTEXTPATH:/nacos}
	server.contextPath=/nacos
	server.port=${NACOS_APPLICATION_PORT:8848}
	spring.datasource.platform=${SPRING_DATASOURCE_PLATFORM:""}
	nacos.cmdb.dumpTaskInterval=3600
	nacos.cmdb.eventTaskInterval=10
	nacos.cmdb.labelTaskInterval=300
	nacos.cmdb.loadDataAtStart=false
	db.num=${MYSQL_DATABASE_NUM:1}
	db.url.0=jdbc:mysql://${MYSQL_SERVICE_HOST}:${MYSQL_SERVICE_PORT:3306}/${MYSQL_SERVICE_DB_NAME}?${MYSQL_SERVICE_DB_PARAM:characterEncoding=utf8&connectTimeout=1000&socketTimeout=3000&autoReconnect=true}
	db.url.1=jdbc:mysql://${MYSQL_SERVICE_HOST}:${MYSQL_SERVICE_PORT:3306}/${MYSQL_SERVICE_DB_NAME}?${MYSQL_SERVICE_DB_PARAM:characterEncoding=utf8&connectTimeout=1000&socketTimeout=3000&autoReconnect=true}
	db.user=${MYSQL_SERVICE_USER}
	db.password=${MYSQL_SERVICE_PASSWORD}
	### The auth system to use, currently only 'nacos' is supported:
	nacos.core.auth.system.type=${NACOS_AUTH_SYSTEM_TYPE:nacos}
	
	
	### The token expiration in seconds:
	nacos.core.auth.default.token.expire.seconds=${NACOS_AUTH_TOKEN_EXPIRE_SECONDS:18000}
	
	### The default token:
	nacos.core.auth.default.token.secret.key=${NACOS_AUTH_TOKEN:SecretKey012345678901234567890123456789012345678901234567890123456789}
	
	### Turn on/off caching of auth information. By turning on this switch, the update of auth information would have a 15 seconds delay.
	nacos.core.auth.caching.enabled=${NACOS_AUTH_CACHE_ENABLE:false}
	nacos.core.auth.enable.userAgentAuthWhite=${NACOS_AUTH_USER_AGENT_AUTH_WHITE_ENABLE:false}
	nacos.core.auth.server.identity.key=${NACOS_AUTH_IDENTITY_KEY:serverIdentity}
	nacos.core.auth.server.identity.value=${NACOS_AUTH_IDENTITY_VALUE:security}
	server.tomcat.accesslog.enabled=${TOMCAT_ACCESSLOG_ENABLED:false}
	server.tomcat.accesslog.pattern=%h %l %u %t "%r" %s %b %D
	# default current work dir
	server.tomcat.basedir=
	## spring security config
	### turn off security
	nacos.security.ignore.urls=${NACOS_SECURITY_IGNORE_URLS:/,/error,/**/*.css,/**/*.js,/**/*.html,/**/*.map,/**/*.svg,/**/*.png,/**/*.ico,/console-fe/public/**,/v1/auth/**,/v1/console/health/**,/actuator/**,/v1/console/server/**}
	# metrics for elastic search
	management.metrics.export.elastic.enabled=false
	management.metrics.export.influx.enabled=false
	
	nacos.naming.distro.taskDispatchThreadCount=10
	nacos.naming.distro.taskDispatchPeriod=200
	nacos.naming.distro.batchSyncKeyCount=1000
	nacos.naming.distro.initDataRatio=0.9
	nacos.naming.distro.syncRetryDelay=5000
	nacos.naming.data.warmup=true`

	cm := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%s-default", e.generateName(nacos)),
			Namespace:   nacos.Namespace,
			Labels:      labels,
			Annotations: nacos.Annotations,
		},
		Data: data,
	}
	myErrors.EnsureNormal(controllerutil.SetControllerReference(nacos, &cm, e.scheme))
	return &cm
}

func (e *KindClient) buildStatefulsetCluster(nacos *nacosgroupv1alpha1.Nacos, ss *appv1.StatefulSet) *appv1.StatefulSet {

	domain := "cluster.local"
	// 从环境变量中获取domain
	for _, env := range nacos.Spec.Env {
		if env.Name == "DOMAIN_NAME" && env.Value != "" {
			domain = env.Value
		}
	}
	ss.Spec.ServiceName = e.generateHeadlessSvcName(nacos)
	serivce := ""
	serivceNoPort := ""
	for i := 0; i < int(*nacos.Spec.Replicas); i++ {
		serivce = fmt.Sprintf("%v%v-%d.%v.%v.%v.%v:%v ", serivce, e.generateName(nacos), i, e.generateHeadlessSvcName(nacos), nacos.Namespace, "svc", domain, NACOS_PORT)
		serivceNoPort = fmt.Sprintf("%v%v-%d.%v.%v.%v.%v ", serivceNoPort, e.generateName(nacos), i, e.generateHeadlessSvcName(nacos), nacos.Namespace, "svc", domain)
	}
	serivce = serivce[0 : len(serivce)-1]
	env := []v1.EnvVar{
		{
			Name:  "NACOS_SERVERS",
			Value: serivce,
		},
	}
	ss.Spec.Template.Spec.Containers[0].Env = append(ss.Spec.Template.Spec.Containers[0].Env, env...)
	// 先检查域名解析再启动
	ss.Spec.Template.Spec.Containers[0].Command = []string{"sh", "-c", fmt.Sprintf("%s&&bin/docker-startup.sh", fmt.Sprintf(initScrit, serivceNoPort))}
	return ss
}

func (e *KindClient) buildHeadlessServiceCluster(svc *v1.Service, nacos *nacosgroupv1alpha1.Nacos) *v1.Service {
	svc.Spec.ClusterIP = "None"
	svc.Name = e.generateHeadlessSvcName(nacos)
	//nacos pod间raft 探测交互走ipv4
	var ipf = make([]v1.IPFamily, 0)
	ipf = append(ipf, v1.IPv4Protocol)
	svc.Spec.IPFamilies = ipf
	var ipPli = v1.IPFamilyPolicySingleStack
	svc.Spec.IPFamilyPolicy = &ipPli
	return svc
}
