# Kubernetes Nacos K8SNacos

本项目包含一个可构建的Nacos Docker Image,旨在利用StatefulSets在[Kubernetes](https://kubernetes.io/)上部署[Nacos](https://nacos.io)



# 已知限制

* 暂时不支持动态增量扩容
* 必须使用持久卷,本项目实现NFS持久卷的例子,如果使用emptyDirs可能会导致数据丢失



# Docker 镜像

在[build](https://github.com/paderlol/nacos-k8s/tree/master/build)目录中包含了已经打好包的Nacos(基于**develop**分支,已提PR,目前的release版本都不支持k8s集群)项目包,以及镜像制作文件,镜像基础环境Ubuntu 16.04、Open JDK 1.8(JDK 8u111).目前镜像已经提交到[Docker Hub](https://hub.docker.com/)。



# 项目目录

| 目录   | 描述                                |
| ------ | ----------------------------------- |
| build  | 构建Nacos镜像的项目包以及Dockerfile |
| deploy | k8s部署yaml文件                     |
| Initdb | Nacos 集群数据库初始化SQL脚本       |



# 使用指南



## 前提要求

* 本项目的使用,是基于你已经对Kubernetes有一定的认知,所以对如何搭建K8S集群,请自行google或者百度
* NFS安装方面也不是本文的重点,请自行google或者百度



## 环境准备

* 机器配置(作者演示使用阿里云ECS)

| 机器内网IP  | 主机名     | 机器配置                                                     |
| ----------- | ---------- | ------------------------------------------------------------ |
| 172.17.79.3 | k8s-master | CentOS Linux release 7.4.1708 (Core) 单核 内存4G 普通云盘40G |
| 172.17.79.4 | node01     | CentOS Linux release 7.4.1708 (Core) 单核 内存4G 普通云盘40G |
| 172.17.79.5 | node02     | CentOS Linux release 7.4.1708 (Core) 单核 内存4G 普通云盘40G |

* Kubernetes 版本：**1.12.2** （如果你和我一样只使用了三台机器,那么记得开启master节点的部署功能）
* NFS 版本：**4.1** 在k8s-master进行安装Server端,并且指定共享目录,本项目指定的**/data/nfs-share**
* Git

 

## 搭建步骤

### Clone项目

在每台机器上都Clone本工程,演示工程就是导入根目录,所以部署路径都是root/nacos-k8s

```shell
git clone https://github.com/paderlol/nacos-k8s.git
```



### 部署数据库

数据库是以指定节点的方式部署,主库部署在node01节点,从库部署在node02节点.

* 部署主库

```shell
#进入clone下来的工程根目录
cd nacos-k8s 
# 在k8s上创建mysql主库
kubectl create -f deploy/mysql/mysql.yml
```



* 部署备库

```shell
#进入clone下来的工程根目录
cd nacos-k8s 
# 在k8s上创建mysql备库
kubectl create -f deploy/mysql/mysql-bak.yml
```



**注意**：如果工程不是导入机器的根目录,那么同样需要修改mysql.yaml和mysql-bak.yaml中挂载路径,因为数据库PVC使用的是本地卷,请注意更改配置中的**path**路径如下所示

```yaml
apiVersion: v1
kind: PersistentVolume
metadata:
  name: mysql-pv-volume
  labels:
    type: local
spec:
  storageClassName: manual
  capacity:
    storage: 20Gi
  accessModes:
    - ReadWriteOnce
  hostPath:
    path: "/root/nacos-k8s/mysql"
---
....其他配置
---
apiVersion: v1
kind: PersistentVolume
metadata:
  name: mysql-init-pv-volume
  labels:
    type: local
spec:
  storageClassName: initdb
  capacity:
    storage: 20Gi
  accessModes:
    - ReadWriteOnce
  hostPath:
    path: "/root/nacos-k8s/initdb"
```



* 部署后查看数据库是否已经正常运行

```shell
#查看主库是否正常运行
kubectl get pod -l app=mysql
NAME                         READY   STATUS    RESTARTS   AGE
mysql-bak-5c5b5bd479-922zv   1/1     Running   0          2d23h
#查看备库是否正常运行
kubectl get pod -l app=mysql-bak
```



### 部署NFS

* 创建角色 K8S在1.6以后默认开启了RBAC

```shell
kubectl create -f deploy/nfs/rbac.yaml
```

提示：如果你的K8S命名空间不是默认"default",那么在创建RBAC之前先执行以下脚本

```shell
# Set the subject of the RBAC objects to the current namespace where the provisioner is being deployed
$ NS=$(kubectl config get-contexts|grep -e "^\*" |awk '{print $5}')
$ NAMESPACE=${NS:-default}
$ sed -i'' "s/namespace:.*/namespace: $NAMESPACE/g" ./deploy/nfs/rbac.yaml

```



* 创建ServiceAccount 以及部署NFS-Client Provisioner

```shell
kubectl create -f deploy/nfs/deployment.yaml
```



* 创建NFS StorageClass

```shell
kubectl create -f deploy/nfs/class.yaml
```



* 查看NFS是否运行正常

```shell
kubectl get pod -l app=nfs-client-provisioner
```





### 部署Nacos 



* 获取主库从库在K8S的地址

```shell
# 查看主库和从库的cluster ip
kubectl get svc

mysql            NodePort    10.105.42.247   <none>        3306:31833/TCP   2d23h
mysql-bak        NodePort    10.105.35.138   <none>        3306:31522/TCP   2d23h
```



* 修改配置文件**depoly/nacos/nacos-pvc-nfs.yaml**,找到如下配置,填入上一步查到的主库和从库地址

```yaml
  db.host.zero: "主库地址"
  db.name.zero: "nacos_devtest"
  db.port.zero: "3306"
  db.host.one: "备库地址"
  db.name.one: "nacos_devtest"
  db.port.one: "3306"
  db.user: "nacos"
  db.password: "nacos"
```



* 创建并运行Nacos集群

``` shell
kubectl create -f nacos-k8s/deploy/nacos/nacos-pvc-nfs.yaml
```



* 查看是否运行正常

```shell
kubectl get pod -l app=nacos


AME      READY   STATUS    RESTARTS   AGE
nacos-0   1/1     Running   0          19h
nacos-1   1/1     Running   0          19h
nacos-2   1/1     Running   0          19h
```







## 测试

### 服务注册

```shell
curl -X PUT 'http://集群地址:8848/nacos/v1/ns/instance?serviceName=nacos.naming.serviceName&ip=20.18.7.10&port=8080'
```



### 服务发现

```shell
curl -X GET 'http://集群地址:8848/nacos/v1/ns/instances?serviceName=nacos.naming.serviceName'
```



### 配置推送

```shell
curl -X POST "http://集群地址:8848/nacos/v1/cs/configs?dataId=nacos.cfg.dataId&group=test&content=helloWorld"
```



### 配置获取

```shell
curl -X GET "http://集群地址:8848/nacos/v1/cs/configs?dataId=nacos.cfg.dataId&group=test"
```



## 常见问题

Q:如果不想搭建NFS,并且想体验nacos-k8s?

 A:可以跳过部署nfs的步骤,最后创建运行nfs时,使用一下以下方式创建

```shell
kubectl create -f nacos-k8s/deploy/nacos/nacos-quick-start.yaml
```

