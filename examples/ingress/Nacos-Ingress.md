# Nacos K8S 通过Traefik实现反向代理



## 上下文

由于Nacos K8s 实现自动扩容和缩容时不能设置ClusterIp,导致无法失去集群IP以及负载能力,所以通过Kubernetes Ingress来弥补集群以及负载均衡的能力。



## 什么是Ingress

> An API object that manages external access to the services in a cluster, typically HTTP.
>
> Ingress can provide load balancing, SSL termination and name-based virtual hosting. 
>
> — 引自[what is ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/#what-is-ingress)

上面这段话引自官网，大致意思：

通常管理集群内部给外部访问的API服务(API通常指的是HTTP)。Ingress可以提供负载平衡、SSL终止和基于名称的虚拟主机。



目前Ingress 实现比较好的两种:

1.[Nginx Ingress](https://kubernetes.github.io/ingress-nginx/)

2.[Traefik Ingress](https://traefik.io/)



## 部署 Traefik Ingress

```powershell
$ git clone https://github.com/containous/traefik.git
# ll traefik/examples/k8s/
总用量 32
-rw-r--r--. 1 root root 1805 11月  9 16:23 cheese-deployments.yaml
-rw-r--r--. 1 root root  519 11月  9 16:23 cheese-ingress.yaml
-rw-r--r--. 1 root root  509 11月  9 16:23 cheese-services.yaml
-rw-r--r--. 1 root root  504 11月  9 16:23 cheeses-ingress.yaml
-rw-r--r--. 1 root root  978 11月  9 16:23 traefik-deployment.yaml
-rw-r--r--. 1 root root 1128 11月  9 16:23 traefik-ds.yaml
-rw-r--r--. 1 root root  694 11月  9 16:23 traefik-rbac.yaml
-rw-r--r--. 1 root root  466 11月  9 16:43 ui.yaml

$ kubectl create -f traefik/examples/k8s/traefik-rbac.yaml
clusterrole "traefik-ingress-controller" created
clusterrolebinding "traefik-ingress-controller" created

$ kubectl create -f traefik/examples/k8s/traefik-deployment.yaml
serviceaccount "traefik-ingress-controller" created
deployment "traefik-ingress-controller" created
service "traefik-ingress-service" created

$ kubectl get pods --all-namespaces -o wide
NAMESPACE     NAME                                         READY     STATUS    RESTARTS   AGE       IP              NODE
kube-system   traefik-ingress-controller-833033881-1lnlt   1/1       Running   0          10s         10.96.4.23      node0.localdomain
...
```



## 验证Traefik Ingress

```powershell
$ kubectl get service --all-namespaces
kube-system   traefik-ingress-service   10.96.8.191      <nodes>       80:反代监听宿主机端口/TCP,8080:控制台宿主机端口/TCP   3m
```



查看traefik-ingress-service的宿主机端口,在浏览器访问http://宿主机地址:控制台宿主机端口,出现Traefik 控制台页面表示成功![jafrbh291l](/Users/zhanglong/Downloads/jafrbh291l.png)



## 部署Nacos ingress

在k8s中部署在examples/ingress目录下的nacos-traefik.yaml

```powershell
$ cd nacos-k8s/
$ kubectl create -f examples/ingress/nacos-traefik.yaml
```



再次访问Traefik 控制台页面,会发现Nacos集群节点已经自动发现.

![image-20190113162611501](/Users/zhanglong/Library/Application Support/typora-user-images/image-20190113162611501.png)



通过http://宿主机地址:反代监听宿主机端口/nacos，可以访问nacos了。



## 注意事项

* 本文只是给大家提供一个模板,告诉大家Nacos如何在K8s中通过ingress完成集群负载
* 生产中还是要通过域名来进行反向代理,配置host的方式
* 例子中默认使用的是轮询方式进行的负载