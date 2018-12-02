#!/usr/bin/env bash

echo "mysql master startup"
kubectl create -f ./deploy/mysql/mysql-master-local.yaml

echo "mysql slave startup"
kubectl create -f ./deploy/mysql/mysql-slave-local.yaml

echo "nacos quick startup"
kubectl create -f ./deploy/nacos/nacos-quick-start.yamlemptyDirs will possibly result in a loss of data