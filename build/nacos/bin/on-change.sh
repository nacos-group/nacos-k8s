#!/bin/bash


while read -ra LINE; do
    PEERS=("${PEERS[@]}" $LINE)
done

echo "" > "${CLUSTER_CONF}"

if [ ${#PEERS[@]} -eq 1 ]; then

    echo "${PEERS[0]}:$NACOS_SERVER_PORT" > "${CLUSTER_CONF}"
    exit
fi
for peer in "${PEERS[@]}"; do
 echo "${peer}:$NACOS_SERVER_PORT" >> "${CLUSTER_CONF}"
done

echo "on change write peers:"${PEERS}
curl -X PUT 'http://localhost:8848/nacos/v1/ns/instance?serviceName=nacos.naming.serviceName&ip=20.18.7.2&port=8888'

curl -X GET 'http://localhost:8848/nacos/v1/ns/instances?serviceName=nacos.naming.serviceName' | python -m json.tool

curl -X GET "http://localhost:8848/nacos/v1/ns/raft/state" | python -m json.tool