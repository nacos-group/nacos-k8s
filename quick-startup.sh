#!/usr/bin/env bash

echo "mysql mysql startup"
sh deploy/mysql/mysql-init.sh && kubectl create -f ./deploy/mysql/mysql-local.yaml


echo "nacos quick startup"
kubectl create -f ./deploy/nacos/nacos-quick-start.yaml
