#!/bin/bash

function snarl_avail() {
    file=${1}
    echo "FILE IS : ${file}"
    z=$(cat ${file} | grep "Download ")
    success=$(echo ${z} | grep -o "Download complete" | wc -l)
    failed=$(echo ${z} | grep -o "Download FAILED" | wc -l)
    pct=$(echo "$success/($success+$failed)" | bc -l)
    echo "Success: $success, Failure: $failed, Total: $(($success+$failed)), SuccessPct: $pct"
}

export -f snarl_avail

usage() {
    echo "usage: ${0} [option]"
    echo 'options:'
    echo '    -w  Watch availabilty every n-th second. | [FILE][TIME]'
    echo '    -a  Print availabilty. | [FILE]'
}

option="${1}"
case ${option} in
    -w) watch -n ${3} -x bash -c "snarl_avail ${2}";;
    -a) snarl_avail ${2};;
    *) usage; exit 1;;
esac