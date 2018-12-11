# Kubernetes Nacos K8SNacos


This project contains a Nacos Docker image meant to facilitate the deployment of [Nacos](https://nacos.io) on [Kubernetes](https://kubernetes.io/) via StatefulSets.



# Quick Start


## Environment

* Machine configuration

| Intranet IP  | Hostname     | Configuration                                                    |
| ----------- | ---------- | ------------------------------------------------------------ |
| 172.17.79.3 | k8s-master | CentOS Linux release 7.4.1708 (Core) Single-core processor Mem 4G Cloud disk 40G |
| 172.17.79.4 | node01     | CentOS Linux release 7.4.1708 (Core) Single-core processor Mem 4G Cloud disk 40G |
| 172.17.79.5 | node02     | CentOS Linux release 7.4.1708 (Core) Single-core processor Mem 4G Cloud disk 40G |

* Kubernetes 版本：**1.12.2** （如果你和我一样只使用了三台机器,那么记得开启master节点的部署功能）
* NFS 版本：**4.1** 在k8s-master进行安装Server端,并且指定共享目录,本项目指定的**/data/nfs-share**
* Git

 

## Clone Project


```shell
git clone https://github.com/nacos-group/nacos-k8s.git
```



### Simple

> If you want to start Nacos without NFS, but **emptyDirs will possibly result in a loss of data**. as follows:

```shell
chmod +x quick-startup.sh
./quick-startup.sh
```



### Advanced

#### Deploy NFS

* Create Role 

```shell
kubectl create -f deploy/nfs/rbac.yaml
```

> If your K8S namespace is not default, execute the following script before creating RBAC


```shell
# Set the subject of the RBAC objects to the current namespace where the provisioner is being deployed
$ NS=$(kubectl config get-contexts|grep -e "^\*" |awk '{print $5}')
$ NAMESPACE=${NS:-default}
$ sed -i'' "s/namespace:.*/namespace: $NAMESPACE/g" ./deploy/nfs/rbac.yaml

```



* Create ServiceAccount And deploy NFS-Client Provisioner

```shell
kubectl create -f deploy/nfs/deployment.yaml
```



* Create NFS StorageClass

```shell
kubectl create -f deploy/nfs/class.yaml
```



* Verify that NFS is working

```shell
kubectl get pod -l app=nfs-client-provisioner
```

#### Deploy database


* Deploy master

```shell

cd nacos-k8s

kubectl create -f deploy/mysql/mysql-master-nfs.yaml
```



* Deploy slave

```shell

cd nacos-k8s 

kubectl create -f deploy/mysql/mysql-slave-nfs.yaml
```



* Verify that Database is working

```shell
# master
kubectl get pod 
NAME                         READY   STATUS    RESTARTS   AGE
mysql-master-gf2vd                        1/1     Running   0          111m

# slave
kubectl get pod 
mysql-slave-kf9cb                         1/1     Running   0          110m
```

#### Deploy Nacos 



* Get master-slave database cluster IP

```shell
# Get master-slave database cluster IP
kubectl get svc

mysql            NodePort    10.105.42.247   <none>        3306:31833/TCP   2d23h
mysql-bak        NodePort    10.105.35.138   <none>        3306:31522/TCP   2d23h
```



* Modify  **depoly/nacos/nacos-pvc-nfs.yaml**

```yaml
data:
  mysql.master.db.name: "db name"
  mysql.master.port: "master db port"
  mysql.slave.port: "slave db port"
  mysql.master.user: "master db username"
  mysql.master.password: "master db password"
```



* Create Nacos

``` shell
kubectl create -f nacos-k8s/deploy/nacos/nacos-pvc-nfs.yaml
```



* Verify that Nacos is working

```shell
kubectl get pod -l app=nacos


AME      READY   STATUS    RESTARTS   AGE
nacos-0   1/1     Running   0          19h
nacos-1   1/1     Running   0          19h
nacos-2   1/1     Running   0          19h
```



# Limitations

* Persistent Volumes must be used. emptyDirs will possibly result in a loss of data



# Docker Image
Image build source code in  [build](https://github.com/nacos-group/nacos-k8s/tree/master/build) directory,It's comprised of a base Ubuntu 16.04 image using the latest release of the OpenJDK JRE based on the 1.8 JVM (JDK 8u111) and the latest stable release of Nacos,0.5.0,
And already pushed into [Docker Hub](https://hub.docker.com/)



# Project directory

| Directory Name   | Description                                |
| ------ | ----------------------------------- |
| build  | Image build source code |
| deploy | Deploy the required files                     |



# Configuration properties

* nacos-pvc-nfs.yaml or nacos-quick-start.yaml 

| Name                  | Required | Description                                    |
| --------------------- | -------- | --------------------------------------- |
| mysql.master.db.name  | Y       | Master database name                          |
| mysql.master.port     | N       | Master database port                          |
| mysql.slave.port      | N       | Slave database port                         |
| mysql.master.user     | Y       | Master database username                        |
| mysql.master.password | Y       | Master database password                       |
| NACOS_REPLICAS        | Y       | The number of clusters must be consistent with the value of the replicas attribute |
| NACOS_SERVER_PORT     | N       | Nacos port,default:8848                |
| PREFER_HOST_MODE      | Y       | Enable Nacos cluster node domain name support               |



* **nfs** deployment.yaml 

| Name       | Required | Description                     |
| ---------- | -------- | ------------------------ |
| NFS_SERVER | Y       | NFS server address           |
| NFS_PATH   | Y       | NFS server shared directory |
| server     | Y       | NFS server address           |
| path       | Y       | NFS server shared directory |



* mysql yaml 

| Name                       | Required | Description                                                        |
| -------------------------- | -------- | ----------------------------------------------------------- |
| MYSQL_ROOT_PASSWORD        | N       | Root password                                                    |
| MYSQL_DATABASE             | Y       | Database Name                                     |
| MYSQL_USER                 | Y       | Database Username                                     |
| MYSQL_PASSWORD             | Y       | Database Password                                |
| MYSQL_REPLICATION_USER     | Y       | Master-slave replication username                |
| MYSQL_REPLICATION_PASSWORD | Y       | Master-slave replication password                 |
| Nfs:server                 | Y       | NFS server address |
| Nfs:path                   | Y       | NFS server shared path |






## Test

### Service registration


```shell
curl -X PUT 'http://cluster-ip:8848/nacos/v1/ns/instance?serviceName=nacos.naming.serviceName&ip=20.18.7.10&port=8080'
```



### Service discovery

```shell
curl -X GET 'http://cluster-ip:8848/nacos/v1/ns/instances?serviceName=nacos.naming.serviceName'
```



### Publish config

```shell
curl -X POST "http://cluster-ip:8848/nacos/v1/cs/configs?dataId=nacos.cfg.dataId&group=test&content=helloWorld"
```



### Get config

```shell
curl -X GET "http://集群地址:8848/nacos/v1/cs/configs?dataId=nacos.cfg.dataId&group=test"
```



## FAQ

Q:If you don't want to build NFS, and you want to experience the nacos-k8s?

 A:You can skip deploying NFS and use the following script to create Nacos

```shell
kubectl create -f nacos-k8s/deploy/nacos/nacos-quick-start.yaml
```



Q:If NFS is not deployed, how is the database deployed？

A:You can deploy in a local persistent, as follows：

```shell
kubectl create -f nacos-k8s/deploy/mysql/nacos-master-local.yaml
```

