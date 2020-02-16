# Kubernetes Nacos


This project contains a Nacos Docker image meant to facilitate the deployment of [Nacos](https://nacos.io) on [Kubernetes](https://kubernetes.io/) via StatefulSets.

[中文文档](https://github.com/nacos-group/nacos-k8s/blob/master/README-CN.md)

# Quick Start

* **Clone Project**


```shell
git clone https://github.com/nacos-group/nacos-k8s.git
```



* **Simple Start**

> If you want to start Nacos without NFS, but **emptyDirs will possibly result in a loss of data**. as follows:

```shell
cd nacos-k8s
chmod +x quick-startup.sh
./quick-startup.sh
```



* **Testing**

  * **Service registration**

  ```powershell
  curl -X PUT 'http://cluster-ip:8848/nacos/v1/ns/instance?serviceName=nacos.naming.serviceName&ip=20.18.7.10&port=8080'
  ```



  * **Service discovery**

  ```powershell
  curl -X GET 'http://cluster-ip:8848/nacos/v1/ns/instances?serviceName=nacos.naming.serviceName'
  ```



  * **Publish config**

  ```powershell
  curl -X POST "http://cluster-ip:8848/nacos/v1/cs/configs?dataId=nacos.cfg.dataId&group=test&content=helloWorld"
  ```



  * **Get config**

  ```powershell
  curl -X GET "http://cluster-ip:8848/nacos/v1/cs/configs?dataId=nacos.cfg.dataId&group=test"
  ```




# Advanced

> In advanced use, the cluster is automatically scaled and data is persisted, but [PersistentVolumeClaims](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#persistentvolumeclaims) must be deployed. In this example, NFS is used.
>



## Deploy NFS

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



* Create `ServiceAccount` And deploy `NFS-Client Provisioner`

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



## Deploy database


* Deploy mysql

```shell

cd nacos-k8s

kubectl create -f deploy/mysql/mysql-nfs.yaml
```







* Verify that Database is working

```shell

kubectl get pod 
NAME                         READY   STATUS    RESTARTS   AGE
mysql-gf2vd                        1/1     Running   0          111m

```



## Deploy Nacos 





* Modify  **depoly/nacos/nacos-pvc-nfs.yaml**

```yaml
data:
  mysql.db.name: "db name"
  mysql.port: " db port"
  mysql.user: " db username"
  mysql.password: " db password"
```



* Create Nacos

``` shell
kubectl create -f nacos-k8s/deploy/nacos/nacos-pvc-nfs.yaml
```



* Verify that Nacos is working

```shell
kubectl get pod -l app=nacos


NAME      READY   STATUS    RESTARTS   AGE
nacos-0   1/1     Running   0          19h
nacos-1   1/1     Running   0          19h
nacos-2   1/1     Running   0          19h
```





## Scale Testing

* Use [`kubectl exec`](https://kubernetes.io/docs/reference/generated/kubectl/kubectl-commands/#exec) to get the cluster config of the Pods in the `nacos` StatefulSet.

```powershell
for i in 0 1; do echo nacos-$i; kubectl exec nacos-$i cat conf/cluster.conf; done
```

The StatefulSet controller provides each Pod with a unique hostname based on its ordinal index. The hostnames take the form of `<statefulset name>-<ordinal index>`. Because the `replicas` field of the `nacos` StatefulSet is set to `2`, In the cluster file only two nacos address



![k8s](/images/k8s.gif)



* Use kubectl to scale StatefulSets

```bash
kubectl scale sts nacos --replicas=3
```

![scale](/images/scale.gif)



* Use [`kubectl exec`](https://kubernetes.io/docs/reference/generated/kubectl/kubectl-commands/#exec) to get the cluster config of the Pods in the `nacos` StatefulSet after scale StatefulSets

```bash
for i in 0 1 2; do echo nacos-$i; kubectl exec nacos-$i cat conf/cluster.conf; done
```

![get_cluster_after](/images/get_cluster_after.gif)



* Use [`kubectl exec`](https://kubernetes.io/docs/reference/generated/kubectl/kubectl-commands/#exec) to get the **state** of the Pods in the `nacos` StatefulSet after scale StatefulSets

```bash
for i in 0 1 2; do echo nacos-$i; kubectl exec nacos-$i curl GET "http://localhost:8848/nacos/v1/ns/raft/state"; done
```

You can find that the new node has joined the cluster

# Prerequisites

- Kubernetes Node configuration(for reference only)

| Hostname   | Configuration                                                                    |
| ---------- | -------------------------------------------------------------------------------- |                    
| k8s-master | CentOS Linux release 7.4.1708 (Core) Single-core processor Mem 4G Cloud disk 40G |
| node01     | CentOS Linux release 7.4.1708 (Core) Single-core processor Mem 4G Cloud disk 40G |
| node02     | CentOS Linux release 7.4.1708 (Core) Single-core processor Mem 4G Cloud disk 40G |

- Kubernetes version：**1.12.2+** 
- NFS version：**4.1+** 




# Limitations

* Persistent Volumes must be used. emptyDirs will possibly result in a loss of data





# Project directory

| Directory Name   | Description                                |
| ------ | ----------------------------------- |
| plugin | Help Nacos cluster achieve automatic scaling in K8s |
| deploy | Deploy the required files                     |



# Configuration properties

* nacos-pvc-nfs.yaml or nacos-quick-start.yaml 

| Name                  | Required | Description                                    |
| --------------------- | -------- | --------------------------------------- |
| mysql.db.name  | Y       | database name                          |
| mysql.port     | N       | database port                          |
| mysql.user     | Y       | database username                        |
| mysql.password | Y       | database password                       |
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
| Nfs:server                 | Y       | NFS server address |
| Nfs:path                   | Y       | NFS server shared path |


