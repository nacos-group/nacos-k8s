#!/bin/bash


NACOS_APPLICATION_PORT=${NACOS_APPLICATION_PORT:-8848}

while read -ra LINE; do
    PEERS=("${PEERS[@]}" $LINE)
done

if [ ${#PEERS[@]} -eq 1 ]; then

    echo "${PEERS[0]}:$NACOS_APPLICATION_PORT" > "${CLUSTER_CONF}"
    exit
fi
CONTENT=""
for peer in "${PEERS[@]}"; do
 CONTENT="${CONTENT}${peer}:$NACOS_APPLICATION_PORT\n"
done

echo -e "${CONTENT%\\n}" > "${CLUSTER_CONF}"
echo "on change write peers:"${PEERS[@]}
