# nacos-operator

nacos-operator项目，快速在K8s上面部署构建nacos。

## 与nacos-k8s的项目区别
### 优点
- 通过operator快速构建nacos集群，指定简单的cr.yaml文件，既可以实现各种类型的nacos集群(数据库选型、standalone/cluster模式等)
- 增加一定的运维能力，在status中增加对nacos集群状态的检查、自动化运维等(后续扩展更多功能)

## 快速开始
```
# 直接使用helm方式安装operator
helm install nacos-operator ./chart/nacos-operator 

# 如果没有helm, 使用kubectl进行安装, 默认安装在default下面
kubectl apply -f chart/nacos-operator/nacos-operator-all.yaml
```

### 启动单实例，standalone模式
查看cr文件
```
cat config/samples/nacos.yaml
apiVersion: nacos.io/v1alpha1
kind: Nacos
metadata:
  name: nacos
spec:
  type: standalone
  image: nacos/nacos-server:1.4.1
  replicas: 1

# 安装demo standalone模式
kubectl apply -f config/samples/nacos.yaml
```
查看nacos实例
```
kubectl get nacos
NAME    REPLICAS   READY     TYPE         DBTYPE   VERSION   CREATETIME
nacos   1          Running   standalone            1.4.1     2021-03-14T09:21:49Z

kubectl get pod  -o wide
NAME                 READY   STATUS    RESTARTS   AGE    IP               NODE        NOMINATED NODE   READINESS GATES
nacos-0   1/1     Running   0          84s    10.168.247.38    slave-100   <none>           <none>

kubectl get nacos nacos -o yaml
...
status
  conditions:
  - instance: 10.168.247.38
    nodeName: slave-100
    podName: nacos-0
    status: "true"
    type: leader
  phase: Running
  version: 1.4.1
```
清除
```
make demo clear=true
```
### 启动集群模式
```
cat config/samples/nacos_cluster.yaml

apiVersion: nacos.io/v1alpha1
kind: Nacos
metadata:
  name: nacos
spec:
  type: cluster
  image: nacos/nacos-server:1.4.1
  replicas: 3
```
```
# 创建nacos集群
kubectl apply -f config/samples/nacos_cluster.yaml

kubectl get po -o wide
NAME             READY   STATUS    RESTARTS   AGE    IP               NODE         NOMINATED NODE   READINESS GATES
nacos-0          1/1     Running   0          111s   10.168.247.39    slave-100    <none>           <none>
nacos-1          1/1     Running   0          109s   10.168.152.186   master-212   <none>           <none>
nacos-2          1/1     Running   0          108s   10.168.207.209   slave-214    <none>           <none>

kubectl get nacos
NAME    REPLICAS   READY     TYPE      DBTYPE   VERSION   CREATETIME
nacos   3          Running   cluster            1.4.1     2021-03-14T09:33:09Z

kubectl get nacos nacos -o yaml -w
...
status:
  conditions:
  - instance: 10.168.247.39
    nodeName: slave-100
    podName: nacos-0
    status: "true"
    type: leader
  - instance: 10.168.152.186
    nodeName: master-212
    podName: nacos-1
    status: "true"
    type: Followers
  - instance: 10.168.207.209
    nodeName: slave-214
    podName: nacos-2
    status: "true"
    type: Followers
  event:
  - code: -1
    firstAppearTime: "2021-03-05T08:35:03Z"
    lastTransitionTime: "2021-03-05T08:35:06Z"
    message: The number of ready pods is too small[]
    status: false
  - code: 200
    firstAppearTime: "2021-03-05T08:36:09Z"
    lastTransitionTime: "2021-03-05T08:36:48Z"
    status: true
  phase: Running
  version: 1.4.1
```

清除
```
make demo clear=true
```
## 配置
全部参数如下

| 参数 | 描述 | 参考值 |
| --- | --- | --- |
| spec.type | 集群类型 | 目前支持standalone 和 cluster |
| spec.image | 镜像地址，兼容社区镜像 | nacos/nacos-server:1.4.1 |
| spec.mysqlInitImage | mysql数据初始镜像地址，mysql模式下将自动导入数据库 | registry.cn-hangzhou.aliyuncs.com/shenkonghui/mysql-client |
| spec.replicas | 实例数量 | 1 |
| spec.database.type | 数据库类型 | 目前支持mysql和embedded |
| spec.database.mysqlHost | mysql连接地址 | 默认mysql |
| spec.database.mysqlPort | mysql端口 | 默认3306 |
| spec.database.mysqlUser | mysql用户 | 默认root |
| spec.database.mysqlPassword | mysql密码 | 默认123456 |
| spec.database.mysqlDb | mysq数据库 | 默认nacos |
| spec.volume.enabled | 是否开启数据卷 | true，如果数据库类型是embedded，请开启数据卷，否则重启pod数据丢失 |
| spec.volume.requests.storage | 存储大小 | 1Gi |
| spec.volume.storageClass | 存储类 | default |
| spec.config | 其他自定义配置，自动映射到custom.propretise | 格式和configmap兼容 |
| spec.k8sWrapper | 支持通用k8配置，即PodSpec对象，会自动覆盖所有内部pod对象 | 无 |

更多配置案例见./config/samples

### 设置模式
目前支持standalone和cluster模式

通过配置spec.type 为 standalone/cluster

### 数据库配置
embedded数据库
```
apiVersion: nacos.io/v1alpha1
kind: Nacos
metadata:
  name: nacos
spec:
  type: standalone
  image: nacos/nacos-server:1.4.1
  replicas: 1
  database:
    type: embedded
  # 启动数据卷，不然重启后数据丢失
  volume:
    enabled: true
    requests:
      storage: 1Gi
    storageClass: default
```

mysql数据库

该模式下需要提供外部mysql连接信息，会自动创建创建nacos数据库，并执行初始化sql
```
apiVersion: nacos.io/v1alpha1
kind: Nacos
metadata:
  name: nacos
spec:
  type: standalone
  image: nacos/nacos-server:1.4.1
  replicas: 1
  database:
    type: mysql
    mysqlHost: mysql
    mysqlDb: nacos
    mysqlUser: root
    mysqlPort: "3306"
    mysqlPassword: "123456"
```
### 自定义配置
1. 通过环境变量配置 兼容nacos-docker项目， https://github.com/nacos-group/nacos-docker
   
    ```
    apiVersion: nacos.io/v1alpha1
    kind: Nacos
    metadata:
      name: nacos
    spec:
      type: standalone
      env:
      - name: JVM_XMS
        value: 2g
      - name: JVM_XMX
        value: 2g
    ```

2. 通过properties文件配置

   https://github.com/nacos-group/nacos-docker/blob/master/build/bin/docker-startup.sh
   
   ```
   export CUSTOM_SEARCH_NAMES="application,custom"
   export CUSTOM_SEARCH_LOCATIONS=${BASE_DIR}/init.d/,file:${BASE_DIR}/conf/
   ```

    支持自定义配置文件，spec.config 会直接映射成custom.properties文件

    ```
    apiVersion: nacos.io/v1alpha1
    kind: Nacos
    metadata:
      name: nacos
    spec:
    ...
      config:|
        management.endpoints.web.exposure.include=*
    ```

## 开发文档
```
# 安装crd
make install

# 以源码方式运行operator
make run

# 编译operator镜像
make image_operator IMG=<your image repo>
```


## FAQ
1. 设置readiness和liveiness集群出问题

    最后一个实例无法ready，搜索了下issus，发现需要以下设置
    ```
    nacos.naming.data.warmup=false
    ```
    
    设置了以后发现，pod能够running，但是集群状态始终无法同步，不同节点出现不同leader；所以暂时不开启readiness和liveiness
   

2. 组集群失败
```
[root@nacos-0 logs]# tail -n 200 nacos.log
java.lang.IllegalStateException: unable to find local peer: nacos-1.nacos-headless.shenkonghui.svc.cluster.local:8848, all peers: []```

```
```
[root@nacos-0 logs]# tail -n 200 alipay-jraft.log
2021-03-16 14:08:48,223 WARN Channel in TRANSIENT_FAILURE state: nacos-2.nacos-headless.shenkonghui.svc.cluster.local:7848.

2021-03-16 14:08:48,223 WARN Channel in SHUTDOWN state: nacos-2.nacos-headless.shenkonghui.svc.cluster.local:7848.
```

```
[root@nacos-0 logs]# tail -n 200 nacos-cluster.log
2021-03-16 14:08:05,710 INFO Current addressing mode selection : FileConfigMemberLookup

2021-03-16 14:08:05,717 ERROR nacos-XXXX [serverlist] failed to get serverlist from disk!, error : The IPv4 address("nacos-2.nacos-headless.shenkonghui.svc.cluster.local") is incorrect.
```
看样子应该是pod是按照顺序启动，无法解析后面还未就绪的pod的ip.
1. 在service中加入属性PublishNotReadyAddresses=true(已实现)。但是如果pod还未分配IP？还是会失败。 
2. 设置statefulset spec.PodManagementPolicy=Parallel(已实现)，让pod同时启动而不是1个1个启动。提高成功率。
3. 加上initcontainer, 检测headless service全部通过以后才能启动pod(已实现)
为了兼容社区docker/同时不想加initcontainer增加复杂度，cmd中更改启动脚本，在启动docker-startup.sh前先执行
```
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
```