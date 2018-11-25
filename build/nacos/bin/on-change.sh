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
