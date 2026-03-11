# @name: Clock Server
# @desc: Print current time every second
# @type: service
# @dir: /tmp
# @env: TZ=UTC
# @env: LANG=en_US.UTF-8
# @pty: true

while true; do date; sleep 1; done
