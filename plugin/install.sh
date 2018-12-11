#! /bin/bash
WORKDIR_VOLUME="/home/nacos/plugins/peer-finder"

echo installing config scripts into "${WORKDIR_VOLUME}"

mkdir -p "${WORKDIR_VOLUME}"
cp /on-start.sh "${WORKDIR_VOLUME}"/
cp /on-change.sh "${WORKDIR_VOLUME}"/
cp /plugin.sh "${WORKDIR_VOLUME}"/
cp /peer-finder "${WORKDIR_VOLUME}"/


