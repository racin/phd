#!/bin/bash

# Global definitions
TOTALPODS=1000
SNARLPATH="\$HOME/snarl/snarl"
SWARMTOOLSPATH="\$HOME/GolangSnippets/swarmtools/listchunks"
#SNARLPATH="\$HOME/go/src/github.com/racin/snarl/snarl"
#START_SWARM_CMD="/Users/racin/go/src/github.com/racin/swarm-racin/build/bin/swarm --bzzaccount=\$HOME/testkey.txt --bzznetworkid=300 \
#--bzzapi=http://localhost:8500 --bzznetworkid=300 --verbosity=0 --no-sync"
CDB="\$HOME/.ethereum/swarm/bzz-4dc45128b557fd44cb6ffb4a79a15fd6646f7d74/"
PV_ROOT="/mnt/snarlbee"

CHUNKDBPATH="--chunkdbpath="$CDB"chunks/"
SNARLDBPATH="--snarldbpath=\$HOME/snarlDBPath/"
#IPCPATH="--ipcpath=/Users/racin/Library/Ethereum/bzzd.ipc"
IPCPATH="--ipcpath=\$HOME/.ethereum/bzzd.ipc"
RM_CMD="rm -rf "$CDB"*"
BBCHAIN1HOME="/home/stud/racin/"

# Function imports
source eval_funcs_podmanage.sh
source eval_funcs_benchmark.sh
source eval_funcs_download.sh
source eval_funcs_pv.sh
source eval_funcs_fileop.sh
source eval_funcs_localswarm.sh
source eval_funcs_bee.sh


function fatal() {
    echo "[$(date +'%Y-%m-%dT%H:%M:%S%z')]: $*" >&2
    exit 1
}

# TODO add checkArgs to all functions.
function checkArgs() {
    [[ "$#" -ne 2 ]] && fatal "Please inform the necessary arguments: ${1}"
}

# Get IP from running pods
function updateIPsFromPods {
BOOTNODEIP=$(eval kubectl get pod swarm-private-bootnode-0 --namespace swarm -o wide | grep -v IP | awk '{print $6}')
SWARMPRIVATEONEIP=$(eval kubectl get pod swarm-private-1 --namespace swarm -o wide | grep -v IP | awk '{print $6}')

# --lightnode=true  -- Not needed.
START_SWARM_CMD="\$HOME/swarm-racin/build/bin/swarm --bzzaccount=\$HOME/testkey.txt --bzznetworkid=300 \
--bzzapi=http://localhost:8500 --maxpeers=1002 --bootnodes=enode://846c424961adc146d54861bdf1eb6015e6908b689fd12d01c61307fffc\
848c22e514f5c898dc9243fbb17aa80750b556772599d84fe86a4b715f40ebc4c049bf@${BOOTNODEIP}:30399 --nat=ip:10.244.1.1 \
--store.cache.size=999999 --store.size=999999 --verbosity=0 --lightnode=false --httpaddr=0.0.0.0"

# --lightnode=true -- Not needed.
SWARM_UPLOAD_CMD="\$HOME/swarm-racin/build/bin/swarm --bzzaccount=\$HOME/testkey.txt --bzznetworkid=300 \
--bzzapi=http://${SWARMPRIVATEONEIP}:8500 --maxpeers=1002 --bootnodes=enode://846c424961adc146d54861bdf1eb6015e6908b689fd12d01c61307fffc\
848c22e514f5c898dc9243fbb17aa80750b556772599d84fe86a4b715f40ebc4c049bf@${BOOTNODEIP}:30399 --nat=ip:10.244.1.1 \
--store.cache.size=999999 --store.size=999999 --verbosity=3 --lightnode=false --httpaddr=0.0.0.0 up --progress "

echo $START_SWARM_CMD
}

# Initalization for Snarl evaluation.
#updateIPsFromPods
#ONLINENODES=$(cat ONLINENODES)
#FAILEDNODES=$(cat FAILEDNODES)

usage() {
    echo "usage: ${0} [option]"
    echo 'options:'
    echo '    -gr  Generate testfiles | [SIZE]+[NAME]+[VERBOSE]'
    echo '    -g   Generate and upload testfiles | [SIZE]+[VERBOSE]+[KILL SWARM]'
    echo '    -bdk Run download benchmark with KNOWN(uploaded) file | [ITERATIONS]+[HASH IDS]+[EXPECTED SHA256]+[NUMPEERS]+[FAILEDNODES]+[FAILRATE]+[REPLICATION]'
    echo '    -bdr Run download benchmark with RANDOM file | [ITERATIONS]+[SIZE]+[NUMPEERS]+[FAILEDNODES]+[FAILRATE]+[REPLICATION]'
    echo '    -e   Entangle file | [PATH]'
    echo '    -f   Randomly fail nodes | [NUMNODES] [TOTALNODES]'
    echo '    -fd  Fail specific nodes. Comma separated | [TOTALNODES] [TOFAIL]'
    echo '    -hcs Change syncing status in swarm pods. | [TRUE/FALSE]'
    echo '    -hon Change running mode of swarm pods (On-True, Off-False). | [TRUE/FALSE]'
    echo '    -term Terminate all pods (Blocks until complete) (Checks every 10 second). | [ATTEMPTS]'
    echo '    -resres Terminates all pods, then restores storage from backup, then starts all pods again. | [REPLICATION]'
    echo '    -bc  Backup chunks on all pods'
    echo '    -rc  Restore chunks on all pods'
    echo '    -cpv  Create PV directories. | [NUMPVS]'
    echo '    -creppv  Create PV replication directories. | [NUMPVS]'
    echo '    -dpv  Delete PV direcotires.'
    echo '    -bpv  Backup PV directories. | [NUMPVS]'
    echo '    -rpv  Restore PV directories (Deprecated). | [NUMPVS]'
    echo '    -qrpv Quickly restore PV directories on all nodes. | [SUFFIX]'
    echo '    -copypvdir Copy replication PV directories from bbchain2 to other nodes | [NUMPVS] [REPLICATION]'
    echo '    -renamepvdir Renames the uploaded replication PV directores from swarm-private-X to pv-X-repY. | [REPLICATION]'
    echo '    -renamesuffixpvdir Renames the uploaded replication PV directores from pv-X-[OLDSUFFIX] to pv-X-[NEWSUFFIX]. | [NEWSUFFIX] [OLDSUFFIX]'
    echo '    -bhomepv  Backup PV directories to HOME. | [NUMPVS]'
    echo '    -diffpv  Run diff on pv and pv-backup directories. | [NUMPVS]'
    echo '    -gcl  Generate chunk list | [FILES OR HASHES] [HASH IDS IF FILE]'
    echo '    -connpeers  Connect local swarm node to pods | [NUMPEERS]'
    echo '    -setvars  Set variables for file hashes. Call with source! | [VARID]'
    echo '    -gea  Get Ethereum Addresses for Bee nodes | [NUMPODS]'
    echo '    -bdss  Delete Bee State store | [NUMPODS]'
    echo '    -bda  Delete Bee everything stored | [NUMPODS]'
    echo
}

option="${1}"
case ${option} in
    -gr) generate_randomfile "${2}" "${3}" "${4}";;
    -g) generate_and_upload_testfile "${2}" "${3}" "${4}";;
    -bdk) run_benchmark_download_known_file "${2}" "${3}" "${4}" "${5}" "${6}" "${7}" "${8}";;
    -bdr) run_benchmark_download_random_file "${2}" "${3}" "${4}" "${5}" "${6}" "${7}";;
    -e) entangle_file "${2}";;
    -f) fail_nodes_random "${2}" "${3}";;
    -fd) fail_nodes_deterministic "${2}" "${3}";;
    -hcs) helm_change_sync "${2}";;
    -hon) helm_change_running_status "${2}";;
    -term) terminatePods "${2}";;
    -resres) restartAndRestorePodsFromBackup "${2}";;
    -bc) backupChunks;;
    -rc) restoreChunks;;
    -cpv) create_pv_directories "${2}";;
    -creppv) create_pv_rep_directories "${2}";;
    -dpv) delete_pv_directories;;
    -bpv) backup_pv_directories "${2}";;
    -rpv) restore_pv_directories "${2}";;
    -qrpv) quick_restore_pv_directories "${2}";;
    -copypvdir) copy_replication_pv_directories "${2}" "${3}";;
    -renamepvdir) rename_replication_pv_directories "${2}";;
    -renamesuffixpvdir) rename_suffix_replication_pv_directories "${2}" "${3}";;
    -bhomepv) backuptohome_pv_directories "${2}";;
    -diffpv) diff_pv_directories "${2}";;
    -gcl) generatechunklist "${2}" "${3}";;
    -connpeers) connect_to_peers "${2}" "true";;
    -setvars) set_variables "${2}";;
    -gea) getEthAddress "${@:2}";;
    -bdss) bee_delete_statestore "${@:2}";;
    -bda) bee_delete_all "${@:2}";;
    *) usage; exit 1;;
esac
