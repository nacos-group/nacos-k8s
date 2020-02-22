#!/usr/bin/env bash
PEER_FINDER_DIR="/home/nacos/plugins/peer-finder"
cd ${PEER_FINDER_DIR} ./peer-finder -on-start=${PEER_FINDER_DIR}/on-start.sh -on-change=${PEER_FINDER_DIR}/on-change.sh -service=${SERVICE_NAME} -domain=${DOMAIN_NAME} || exit &
