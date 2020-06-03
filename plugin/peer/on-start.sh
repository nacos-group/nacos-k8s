#!/bin/bash


NACOS_APPLICATION_PORT=${NACOS_APPLICATION_PORT:-8848}

while read -ra LINE; do
    PEERS=("${PEERS[@]}" $LINE)
done

echo "" > "${CLUSTER_CONF}"


for peer in "${PEERS[@]}"; do
 echo "${peer}:$NACOS_APPLICATION_PORT" >> "${CLUSTER_CONF}"
done


