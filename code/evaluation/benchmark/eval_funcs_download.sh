# Param1: Test iterations, Param2: HashIDS, Param3: Hash output
# Param4: numPeers, Param5: failednodes, Param6: failrate, Param7: Replication(To restore from)
function test_download_snarl {
    local TESTCOUNTER=-1
    local TESTLIMIT=${1}
    
    local COMMAND="download"

    if [[ -z ${2} ]]; then
       echo "Please specify ID parameter"
       exit 1
    fi
    # Split params to get the size.
    local HASHPARAMS=$(echo ${2} | grep -o "," | wc -l)

    if [ $HASHPARAMS == 0 ]; then
        local IDS=${2}
        local SIZE="0"
    elif [ $HASHPARAMS == 4 ]; then
        local SIZE=$(echo ${2} | cut -d "," -f1)
        local DATAID=$(echo ${2} | cut -d "," -f2) # Required param
        local PHOR=$(echo ${2} | cut -d "," -f3)
        local PRIG=$(echo ${2} | cut -d "," -f4)
        local PLEF=$(echo ${2} | cut -d "," -f5)
        local IDS=${SIZE}","${DATAID}","${PHOR}","${PRIG}","${PLEF}
        if [[ ! -z ${3} ]]; then
            local CHUNKSFILTERS="bench_"${3}"/chunks_"${DATAID}",bench_"${3}"/chunks_"${PHOR}",bench_"${3}"/chunks_"${PRIG}",bench_"${3}"/chunks_"${PLEF}
            local CHUNKSFILTERSDATA="bench_"${3}"/chunks_"${DATAID}
            local PARITYTREELEAVES=$(cat bench_"${3}"/chunks_"${PHOR}" | grep -o -P ".:1}." | wc -l)
            local SIZEOFPARITY=$(printf "%x" $((${PARITYTREELEAVES} * 4096)))
            echo "SIZE OF PARITY: ${SIZEOFPARITY}"
            echo "CHUNKFILTERS: "
            echo $CHUNKSFILTERS
        fi
    else
        echo "Invalid number of ID parameters. Must be 1 or 5"
        exit 1
    fi

    echo $IDS
    local FNODES=${5}
    
    local LOGRESULT="download_${SIZE}_${FNODES}_"$(( RANDOM % 10000000 ))$(( RANDOM % 10000000 ))$(( RANDOM % 10000000 ))

    # Benchmark mode
    local BMARK="--benchmark=true"

    # Expected hash output
    if [[ ! -z ${3} ]]; then
        local HOUT="--hashoutput=${3}"
    fi

    # Number of peers that we must be connected to before running tests (Default: 20% of total peers)
    # $(($TOTALPODS / 5))
    if [[ ! -z ${4} ]]; then
        local NPEER="--numPeers=5" # TODO - This should somehow be number of peers that we use as the argument ( ${4} ). Possible this mush be calculated after we finish connecting to peers.
    else
        local NPEER="--numPeers=1"
    fi

    # Failrate: Simulate chunks being unavailable
    if [[ ! -z ${6} ]]; then
        local FRATE="--failrate=${6}"
    else
        local FRATE="--failrate=0"
    fi
    
    # The command to be executed
    # [failednodes] will always be set to -1 as we do not want to simulate node failure on the client side.
    # Instead we will stop/continue the Swarm process on each individual node for each iteration.
    local CMD=$SNARLPATH" "$COMMAND" "$IDS" "$BMARK" "$HOUT" \
    "$NPEER" "$FRATE" --failednodes=-1 "$CHUNKDBPATH" "$SNARLDBPATH" "$IPCPATH" >> "$LOGRESULT".log"
    echo $CMD

    local CMD_DL_D=$SNARLPATH" "$COMMAND" "${SIZE}","$DATAID" "$BMARK" "$HOUT" \
    "$NPEER" "$FRATE" --failednodes=-1 "$CHUNKDBPATH" "$SNARLDBPATH" "$IPCPATH" >> "$LOGRESULT"_Ver_D.log"
    echo $CMD_DL_D
    local CMD_DL_H=$SNARLPATH" "$COMMAND" "${SIZEOFPARITY}","$PHOR" "$BMARK" "$HOUT" \
    "$NPEER" "$FRATE" --failednodes=-1 "$CHUNKDBPATH" "$SNARLDBPATH" "$IPCPATH" >> "$LOGRESULT"_Ver_H.log"
    echo $CMD_DL_H
    local CMD_DL_R=$SNARLPATH" "$COMMAND" "${SIZEOFPARITY}","$PRIG" "$BMARK" "$HOUT" \
    "$NPEER" "$FRATE" --failednodes=-1 "$CHUNKDBPATH" "$SNARLDBPATH" "$IPCPATH" >> "$LOGRESULT"_Ver_R.log"
    local CMD_DL_L=$SNARLPATH" "$COMMAND" "${SIZEOFPARITY}","$PLEF" "$BMARK" "$HOUT" \
    "$NPEER" "$FRATE" --failednodes=-1 "$CHUNKDBPATH" "$SNARLDBPATH" "$IPCPATH" >> "$LOGRESULT"_Ver_L.log"

    # Verify persistence before we start.
    local VERIFYCMD=$SWARMTOOLSPATH" listchunks "$TOTALPODS" --frequency=1 --verboselistpods=true --filters="$CHUNKSFILTERS" --verifypersistence="$CHUNKSFILTERS" --fullPath=false > bench_"${3}"/check_"$LOGRESULT"_INIT.log"
    eval $VERIFYCMD

    # If TESTCOUNTER is -1 => Verify that we can actually retrieve each chunk.
    while [[ $TESTCOUNTER -lt $TESTLIMIT ]]; do
        rm -rf $HOME/snarlDBPath
        eval $RM_CMD & 

        # Delete list of known peers. (Because they might be offline!)
        rm -rf $HOME/.ethereum/swarm/nodes/*
        export CONNTOPEERSFINISH="false"
        
        sleep 20

        # Always test in no-sync mode
        eval $START_SWARM_CMD" --no-sync" &

        # Sleep 30 seconds before attempting to dial peers
        sleep 30


        if [[ $TESTCOUNTER != -1 && ! -z ${FNODES} ]]; then
            # Lets fail some nodes
            fail_nodes_random ${FNODES} $TOTALPODS
            sleep 30
        fi

        # Connect to peers in parallell while verifying. Can read output in $HOME/nohup.out
        connect_to_peers ${4} "true" &
        
        if [[ $TESTCOUNTER != -1 ]]; then
            # Lets verify persistence. Not neccessary to copy the chunks again.
            local VERIFYCMD=$SWARMTOOLSPATH" listchunks "$TOTALPODS" --verboselistpods=true --filters="$CHUNKSFILTERS" --verifypersistence="$CHUNKSFILTERS" --podFilter=ONLINENODES --fullPath=false --copychunks=false > bench_"${3}"/check_"$LOGRESULT"_"$TESTCOUNTER".log"
            echo $VERIFYCMD
            eval $VERIFYCMD

            cat "bench_"${3}"/check_"$LOGRESULT"_"$TESTCOUNTER".log" | grep "\---.*Verifying" | awk -F ' ' '{gsub(/^\[+/, "", $2); gsub(/\]+$/, "", $2); split($9,a,"/chunks_0x"); print $2 " - " a[2]}' > "bench_"${3}"/check_"$LOGRESULT"_DATA_"$TESTCOUNTER".log"
            local MAXCONPEERITERATIONS=240
            local WANTCONPEERS=$((${4}-${FNODES}))
        else
            local MAXCONPEERITERATIONS=24000
            local WANTCONPEERS=${4}
        fi

        # Ensure we are connected to enough peers. Try for 20 minutes (10 sec * 120) - after verifying finishes.
        local CONPEERITERATIONS=0
        
        while [[ $CONPEERITERATIONS -lt $MAXCONPEERITERATIONS ]]; do
            local connPeers=$(/usr/bin/geth --exec "admin.peers.length" attach $HOME/.ethereum/bzzd.ipc)
            let CONPEERITERATIONS=CONPEERITERATIONS+1
            if [[ $connPeers -ge ${WANTCONPEERS} ]]; then
                export CONNTOPEERSFINISH="true"
                echo "Successfully connected to $connPeers peers. Wanted ${WANTCONPEERS} peers."
                break
            else 
                if [[ $CONPEERITERATIONS -ge $MAXCONPEERITERATIONS ]]; then
                    echo "UNABLE TO CONNECT TO ${WANTCONPEERS} peers. Aborting test iteration. (Only connected to $connPeers)"
                    break 2
                fi
                echo "Want ${WANTCONPEERS} peers. Is now connected to ${connPeers} peers."
            fi
            sleep 10
        done
        
        if [[ $TESTCOUNTER != -1 ]]; then
            eval $CMD
        else
            eval $CMD_DL_D
            eval $CMD_DL_H
            eval $CMD_DL_R
            eval $CMD_DL_L
        fi

        # Terminate Swarm process.
        local SWARMPID=$(ps x | grep [s]warm | grep -v kubectl | grep -v echo | grep -v snarl | awk '{print $1}')
        kill -9 $SWARMPID 

        sleep 20

        let TESTCOUNTER=TESTCOUNTER+1

        # All Swarm pods: Restart and restore from backup. (Except on the last iteration.)
        if [[ $TESTCOUNTER -lt $TESTLIMIT ]]; then
            restartAndRestorePodsFromBackup ${7}
        fi

        if [[ $TERMINATEPODSRESULT == "ERROR" ]]; then
            echo "ERROR RESTARTING PODS. SHUTTING DOWN THE TEST."
            break
        fi
    done
}