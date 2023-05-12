#!/bin/sh
option="${1}"

stop_process() {
	pkill -STOP bee &>/dev/null
}

continue_process() {
	pkill -CONT bee &>/dev/null
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
