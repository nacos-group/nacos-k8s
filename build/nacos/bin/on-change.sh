#!/bin/bash

while read -ra LINE; do
    PEERS=("${PEERS[@]}" $LINE)
done


echo ${PEERS}
