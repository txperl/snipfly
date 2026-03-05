#!/usr/bin/env bash
# @name: slow-task
# @type: oneshot
# @desc: Long-running oneshot (test stop with Space key)

for i in $(seq 1 30); do
    echo "Step $i/30..."
    sleep 1
done
echo "Done!"
