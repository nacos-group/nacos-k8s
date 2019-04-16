## 使用ceph存储部署nacos

### 下载external-storage
```shell
git clone https://github.com/kubernetes-incubator/external-storage.git
```

### 安装cephfs-provisioner
```shell
cd external-storage/ceph/cephfs/deploy/
NAMESPACE=nacos
sed -r -i "s/namespace: [^ ]+/namespace: $NAMESPACE/g" ./rbac/*.yaml
sed -r -i "N;s/(name: PROVISIONER_SECRET_NAMESPACE.*\n[[:space:]]*)value:.*/\1value: $NAMESPACE/" ./rbac/deployment.yaml
kubectl -n $NAMESPACE apply -f ./rbac
```

如遇到问题，rbd挂载后报错`Input/output`，需要添加参数 `- '-disable-ceph-namespace-isolation=true'`
[问题链接](https://github.com/kubernetes-incubator/external-storage/issues/345#issuecomment-414892515)

### 安装ceph rbd-provisioner
```shell
cd external-storage/ceph/rbd/deploy/
NAMESPACE=nacos
sed -r -i "s/namespace: [^ ]+/namespace: $NAMESPACE/g" ./rbac/clusterrolebinding.yaml ./rbac/rolebinding.yaml
kubectl -n $NAMESPACE apply -f ./rbac
```

*ceph集群 cephfs和rbd创建略过*

### 创建ceph secret
```shell
ceph auth list # ceph 服务器上查看key
# mysql secret type需要设为kubernetes.io/rbd
kubectl create secret generic ceph-secret-admin --from-literal=key='AQBTTLRcKesZGxAABYIX6GwiiBooyJ9Jxxxxxx==' --namespace=nacos
kubectl create secret generic ceph-secret-mysql --from-literal=key='AQBTTLRcKesZGxAABYIX6GwiiBooyJ9Jyyyyyy==' --type=kubernetes.io/rbd --namespace=nacos
kubectl create secret generic ceph-secret-mysql-slave --from-literal=key='AQBTTLRcKesZGxAABYIX6GwiiBooyJ9Jzzzzzz==' --type=kubernetes.io/rbd --namespace=nacos
```

### 创建sc和pvc
```shell
kubectl -n nacos apply -f deploy/ceph/sc.yaml
kubectl -n nacos apply -f deploy/ceph/pvc.yaml
```

### 安装mysql,mysql-slave
```shell
kubectl -n nacos apply -f deploy/mysql/mysql-master-ceph.yaml
kubectl -n nacos apply -f deploy/mysql/mysql-slave-ceph.yaml
```

### 安装nacos
```shell
kubectl -n nacos apply -f deploy/nacos/nacos-pvc-ceph.yaml
```

### 已知问题
  - 启动后需要重启pod nacos-0集群才能正常使用
