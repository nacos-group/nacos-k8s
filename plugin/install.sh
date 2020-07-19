#! /bin/bash
WORKDIR_VOLUME="/home/nacos/plugins/peer-finder"

echo installing config scripts into "${WORKDIR_VOLUME}"

mkdir -p "${WORKDIR_VOLUME}"
cp -f /on-start.sh "${WORKDIR_VOLUME}"/
cp -f /on-change.sh "${WORKDIR_VOLUME}"/
cp -f /plugin.sh "${WORKDIR_VOLUME}"/
cp -f /peer-finder "${WORKDIR_VOLUME}"/


