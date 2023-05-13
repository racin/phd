# Param1: NUMPVS
function create_pv_directories {
    checkArgs ${1} "number_of_pvs"
    local ITERATIONS=0
    while [[ $ITERATIONS -lt ${1} ]]; do
        bbnode=$((${ITERATIONS} % 28 + 3))
        #echo "PV ${ITERATIONS} goes to BBchain${bbnode}"
        echo "ssh -t bbchain${bbnode} \"mkdir -p ${PV_ROOT}/pv-${ITERATIONS}\""
        
        ssh -t bbchain${bbnode} "mkdir -p ${PV_ROOT}/pv-${ITERATIONS}"
        ssh -t bbchain${bbnode} "mkdir -p ${PV_ROOT}/pv-${ITERATIONS}-backup"
        # ssh -t bbchain${bbnode} "mkdir -p ${PV_ROOT}/pv-${ITERATIONS}-rep1"
        # ssh -t bbchain${bbnode} "mkdir -p ${PV_ROOT}/pv-${ITERATIONS}-rep1_treerep_5"
        # ssh -t bbchain${bbnode} "mkdir -p ${PV_ROOT}/pv-${ITERATIONS}-rep1_treerep_9"
        # ssh -t bbchain${bbnode} "mkdir -p ${PV_ROOT}/pv-${ITERATIONS}-rep2"
        # ssh -t bbchain${bbnode} "mkdir -p ${PV_ROOT}/pv-${ITERATIONS}-rep2_treerep_9"
        let ITERATIONS=ITERATIONS+1
    done
}

function racin {
    local ITERATIONS=0
    while [[ $ITERATIONS -lt 1000 ]]; do
        bbnode=$((${ITERATIONS} % 28 + 3))
        #echo "PV ${ITERATIONS} goes to BBchain${bbnode}"
        echo "ssh -t bbchain${bbnode} \"mkdir -p /home/stud/racin/extra100/pv-${ITERATIONS}-backup\""
        ssh -t bbchain${bbnode} "mkdir -p /home/stud/racin/extra100/pv-${ITERATIONS}-backup"
        #ssh -t bbchain$i chown -R root:root ${PV_ROOT}/
        let ITERATIONS=ITERATIONS+1
    done
}

# Param1: NUMPVS
function create_pv_rep_directories {
    local ITERATIONS=0
    while [[ $ITERATIONS -lt ${1} ]]; do
        bbnode=$((${ITERATIONS} % 28 + 3))
        #echo "PV ${ITERATIONS} goes to BBchain${bbnode}"
        echo "ssh -t bbchain${bbnode} \"mkdir -p ${PV_ROOT}/pv-${ITERATIONS}-rep1\""
        echo "ssh -t bbchain${bbnode} \"mkdir -p ${PV_ROOT}/pv-${ITERATIONS}-rep1_treerep_5\""
        echo "ssh -t bbchain${bbnode} \"mkdir -p ${PV_ROOT}/pv-${ITERATIONS}-rep1_treerep_9\""
        # echo "ssh -t bbchain${bbnode} \"mkdir -p ${PV_ROOT}/pv-${ITERATIONS}-rep1_nonodes\""
        echo "ssh -t bbchain${bbnode} \"mkdir -p ${PV_ROOT}/pv-${ITERATIONS}-rep2\""
        echo "ssh -t bbchain${bbnode} \"mkdir -p ${PV_ROOT}/pv-${ITERATIONS}-rep2_treerep_9\""
        # echo "ssh -t bbchain${bbnode} \"mkdir -p ${PV_ROOT}/pv-${ITERATIONS}-rep2_nonodes\""
        # echo "ssh -t bbchain${bbnode} \"mkdir -p ${PV_ROOT}/pv-${ITERATIONS}-rep3\""
        # echo "ssh -t bbchain${bbnode} \"mkdir -p ${PV_ROOT}/pv-${ITERATIONS}-rep3_nonodes\""
        # echo "ssh -t bbchain${bbnode} \"mkdir -p ${PV_ROOT}/pv-${ITERATIONS}-rep4\""
        # echo "ssh -t bbchain${bbnode} \"mkdir -p ${PV_ROOT}/pv-${ITERATIONS}-rep4_nonodes\""
        # echo "ssh -t bbchain${bbnode} \"mkdir -p ${PV_ROOT}/pv-${ITERATIONS}-rep5\""
        # echo "ssh -t bbchain${bbnode} \"mkdir -p ${PV_ROOT}/pv-${ITERATIONS}-rep5_nonodes\""
        ssh -t bbchain${bbnode} "mkdir -p ${PV_ROOT}/pv-${ITERATIONS}-rep1"
        ssh -t bbchain${bbnode} "mkdir -p ${PV_ROOT}/pv-${ITERATIONS}-rep1_treerep_5"
        ssh -t bbchain${bbnode} "mkdir -p ${PV_ROOT}/pv-${ITERATIONS}-rep1_treerep_9"
        # ssh -t bbchain${bbnode} "mkdir -p ${PV_ROOT}/pv-${ITERATIONS}-rep1_nonodes"
        ssh -t bbchain${bbnode} "mkdir -p ${PV_ROOT}/pv-${ITERATIONS}-rep2"
        ssh -t bbchain${bbnode} "mkdir -p ${PV_ROOT}/pv-${ITERATIONS}-rep2_treerep_9"
        # ssh -t bbchain${bbnode} "mkdir -p ${PV_ROOT}/pv-${ITERATIONS}-rep2_nonodes"
        # ssh -t bbchain${bbnode} "mkdir -p ${PV_ROOT}/pv-${ITERATIONS}-rep3"
        # ssh -t bbchain${bbnode} "mkdir -p ${PV_ROOT}/pv-${ITERATIONS}-rep3_nonodes"
        # ssh -t bbchain${bbnode} "mkdir -p ${PV_ROOT}/pv-${ITERATIONS}-rep4"
        # ssh -t bbchain${bbnode} "mkdir -p ${PV_ROOT}/pv-${ITERATIONS}-rep4_nonodes"
        # ssh -t bbchain${bbnode} "mkdir -p ${PV_ROOT}/pv-${ITERATIONS}-rep5"
        # ssh -t bbchain${bbnode} "mkdir -p ${PV_ROOT}/pv-${ITERATIONS}-rep5_nonodes"
        #ssh -t bbchain$i chown -R root:root ${PV_ROOT}/
        let ITERATIONS=ITERATIONS+1
    done
}


function delete_pv_directories {
    for i in {3..30} ; do
        #echo "Deleting all PVs in bbchain${1}"
        echo "ssh -t bbchain${i} \"rm -rf ${PV_ROOT}/pv-*\""
        ssh -t bbchain${i} "rm -rf ${PV_ROOT}/pv-*"
    done
}

# Param1: NUMPVS
function backup_pv_directories {
    local ITERATIONS=0
    while [[ $ITERATIONS -lt ${1} ]]; do
        bbnode=$((${ITERATIONS} % 28 + 3))
        ssh -t bbchain${bbnode} "rm -rf ${PV_ROOT}/pv-${ITERATIONS}-backup/*"
        ssh -t bbchain${bbnode} "cp -R ${PV_ROOT}/pv-${ITERATIONS}/. ${PV_ROOT}/pv-${ITERATIONS}-backup/"
        echo "cp -R ${PV_ROOT}/pv-${ITERATIONS}/. ${PV_ROOT}/pv-${ITERATIONS}-backup/"
        let ITERATIONS=ITERATIONS+1
    done
}

function racinbackup {
    local ITERATIONS=0
    while [[ $ITERATIONS -lt ${1} ]]; do
        bbnode=$((${ITERATIONS} % 28 + 3))
        ssh -t bbchain${bbnode} "rm -rf /home/stud/racin/extra100/pv-${ITERATIONS}-backup/*"
        ssh -t bbchain${bbnode} "cp -R ${PV_ROOT}/pv-${ITERATIONS}/. /home/stud/racin/extra100/pv-${ITERATIONS}-backup/"
        echo "cp -R ${PV_ROOT}/pv-${ITERATIONS}/. /home/stud/racin/extra100/pv-${ITERATIONS}-backup/"
        let ITERATIONS=ITERATIONS+1
    done
}

function backuptohome_pv_directories {
    local ITERATIONS=0
    while [[ $ITERATIONS -lt ${1} ]]; do
        bbnode=$((${ITERATIONS} % 28 + 3))
        # mkdir -p /home/stud/racin/backup_PV/backup/pv-${ITERATIONS}-backup
        ssh -t bbchain${bbnode} "rm -rf /home/stud/racin/pv-${ITERATIONS}-backup/*"
        ssh -t bbchain${bbnode} "cp -R ${PV_ROOT}/pv-${ITERATIONS}/. /home/stud/racin/pv-${ITERATIONS}-backup/"
        echo "cp -R ${PV_ROOT}/pv-${ITERATIONS}/. /home/stud/racin/pv-${ITERATIONS}-backup/"
        let ITERATIONS=ITERATIONS+1
    done
}

# Param1: Diff actual and backup
function diff_pv_directories {
    local ITERATIONS=0
    while [[ $ITERATIONS -lt ${1} ]]; do
        bbnode=$((${ITERATIONS} % 28 + 3))
        ssh -t bbchain${bbnode} "diff -qur ${PV_ROOT}/pv-${ITERATIONS}/ ${PV_ROOT}/pv-${ITERATIONS}-backup/"
        echo "diff -qur ${PV_ROOT}/pv-${ITERATIONS}/ ${PV_ROOT}/pv-${ITERATIONS}-backup/"
        let ITERATIONS=ITERATIONS+1
    done
}

# Param1: NUMPVS
function restore_pv_directories {
    local ITERATIONS=0
    while [[ $ITERATIONS -lt ${1} ]]; do
        bbnode=$((${ITERATIONS} % 28 + 3))
        ssh -t bbchain${bbnode} "rm -rf ${PV_ROOT}/pv-${ITERATIONS}/*"
        ssh -t bbchain${bbnode} "cp -R ${PV_ROOT}/pv-${ITERATIONS}-backup/. ${PV_ROOT}/pv-${ITERATIONS}/"
        echo "cp -R ${PV_ROOT}/pv-${ITERATIONS}-backup/. ${PV_ROOT}/pv-${ITERATIONS}/"
        let ITERATIONS=ITERATIONS+1
    done
}

# Param1: NUMPVS, Param2: Suffix
function copy_replication_pv_directories {
    local ITERATIONS=0
    while [[ $ITERATIONS -lt ${1} ]]; do
        bbnode=$((${ITERATIONS} % 28 + 3))
        #ssh -t bbchain${bbnode} "rm -rf ${PV_ROOT}/pv-${ITERATIONS}-${2}/*"
        rsync -av --delete "/home/stud/racin/backup_PV/${2}/pv-${ITERATIONS}-${2}" "racin@bbchain${bbnode}:${PV_ROOT}/"
        echo "rsync -av --delete /home/stud/racin/backup_PV/${2}/pv-${ITERATIONS}-${2} racin@bbchain${bbnode}:${PV_ROOT}/"
        let ITERATIONS=ITERATIONS+1
    done
}

# Param1: Replication
function rename_replication_pv_directories {
    # Find all directories that we want to rename
    local allDirs=$(ls -d /home/stud/racin/backup_PV/rep${1}/*)
    for dir in $allDirs
    do
        # Extract number
        local number=$(echo ${dir} | cut -d '-' -f3)
        local private=$(echo ${dir} | cut -d '-' -f2)

        if [[ $private != "private" ]]; then
            echo "Did not find substring private. Doing nothing"
            continue
        fi

        echo "mv ${dir} /home/stud/racin/backup_PV/rep${1}/pv-${number}-rep${1}"
        mv ${dir} "/home/stud/racin/backup_PV/rep${1}/pv-${number}-rep${1}"
    done
}

# Param1: New suffix, Param2: Old suffix
function rename_suffix_replication_pv_directories {
    # Find all directories that we want to rename
    local allDirs=$(ls -d /home/stud/racin/backup_PV/${1}/*)
    for dir in $allDirs
    do
        if [[ $dir != *${2} ]]; then
            echo "Did not find substring ${2} in ${dir}. Doing nothing"
            continue
        fi

        # New directory name is the old suffix (2) replaced with new suffix (1)
        newDir=$(echo ${dir} | sed -r "s/(.*)${2}/\1${1}/")
        
        echo "mv ${dir} ${newDir}"
        mv ${dir} ${newDir}
    done
}

# Param1: Suffix
QUICKRESTORERESULT=""
function quick_restore_pv_directories {
    local CURRNODE=3
    local LASTNODE=30
    while [[ $CURRNODE -le ${LASTNODE} ]]; do
        ssh -t bbchain${CURRNODE} "/home/stud/racin/scripts/node_restoreBackupSwarmState.sh quickrestore ${1}" &>/dev/null &
        let CURRNODE=CURRNODE+1
    done

    # Wait until all jobs finish ...
    local TESTCOUNTER=0
    local TESTLIMIT=30

    while [[ $TESTCOUNTER -lt $TESTLIMIT ]]; do
        local RUN=$(ps aux | grep -i [q]uickrestore | grep -v sudo | wc -l)
        let TESTCOUNTER=TESTCOUNTER+1
        if [[ $RUN -eq 0 ]]; then
            QUICKRESTORERESULT="ALL_RESTORED"
            break
        elif [[ $TESTCOUNTER -eq $TESTLIMIT ]]; then
            QUICKRESTORERESULT="ERROR"
            break
        fi
        echo $RUN" jobs still running."
        sleep 10
    done
}

# backupChunks is not used.
function backupChunks {
    # Stop all swarm pods
    fail_nodes_random $TOTALPODS $TOTALPODS
    sleep 10
    local ITERATIONS=0

    # Copy all chunks to separate dir
    while [[ $ITERATIONS -lt $TOTALPODS ]]; do
        kubectl exec swarm-private-${ITERATIONS} -n swarm -- /bin/sh "/root/restoreBackupSwarmState.sh" "backup" &
        let ITERATIONS=ITERATIONS+1
    done

    # Continue all swarm pods
    fail_nodes_random 0 $TOTALPODS
    sleep 10
}

# restoreChunks is not used.
function restoreChunks {
    # Stop all swarm pods
    fail_nodes_random $TOTALPODS $TOTALPODS
    sleep 10
    local ITERATIONS=0
    
    # Copy all chunks from backup to live directory.
    while [[ $ITERATIONS -lt $TOTALPODS ]]; do
        kubectl exec swarm-private-${ITERATIONS} -n swarm -- /bin/sh "/root/restoreBackupSwarmState.sh" "restore" &
        let ITERATIONS=ITERATIONS+1
    done

     # Continue all swarm pods
    fail_nodes_random 0 $TOTALPODS
    sleep 10
}
