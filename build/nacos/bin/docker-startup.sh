#!/bin/bash
cd /home/nacos/bin
./peer-finder -on-start=/home/nacos/bin/on-start.sh -on-change=/home/nacos/bin/on-change.sh -service=nacos-headless -ns=default

