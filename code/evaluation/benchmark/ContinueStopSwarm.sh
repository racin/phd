#!/bin/bash
PID=$(ps aux | grep -E 'swarm.*@[0-9].*:[0-9]' | cut -d ' ' -f4)
# TODO. Check if pgrep will work in the above command.

option="${1}"

stop_process() {
    kill -STOP $PID &>/dev/null
}

continue_process() {
    kill -CONT $PID &>/dev/null
}

usage() {
    echo "usage: ${0} [option]"
    echo 'options:'
    echo '    -s Stop process'
    echo '    -c Continue process'
    echo
}

case ${option} in
    -s) stop_process;;
    -c) continue_process;;
    *) usage; exit 1;;
esac
# STOP
# CONTINUE