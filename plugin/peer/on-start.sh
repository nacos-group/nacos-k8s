#!/bin/bash


while read -ra LINE; do
    PEERS=("${PEERS[@]}" $LINE)
done

echo "" > "${CLUSTER_CONF}"


for peer in "${PEERS[@]}"; do
 echo "${peer}:$NACOS_SERVER_PORT" >> "${CLUSTER_CONF}"
done


