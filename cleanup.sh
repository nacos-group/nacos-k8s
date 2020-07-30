#!/usr/bin/env bash

echo "mysql master deleting"
kubectl delete -f ./deploy/mysql/mysql-master-local.yaml

echo "mysql slave deleting"
kubectl delete -f ./deploy/mysql/mysql-slave-local.yaml

echo "nacos quick deleting"
kubectl delete -f ./deploy/nacos/nacos-quick-start.yaml
