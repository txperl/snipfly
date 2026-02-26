#!/usr/bin/env bash
# @name: echo-server
# @desc: Print current time every second
# @type: service

echo "Echo server started"
while true; do
    echo "[$(date '+%H:%M:%S')] tick"
    sleep 1
done
