# Param1: Size, Param2: Verbose, Param3: Shut off Swarm
function generate_and_upload_testfile {
    local NAME=$(( RANDOM % 10000000 ))$(( RANDOM % 10000000 ))$(( RANDOM % 10000000 ))
    head -c ${1} </dev/urandom >${NAME}
    local RANDOMPATH=$(pwd)/${NAME}
    RANDOMSHASUM=$(eval shasum -a 256 $(pwd)/${NAME})
    RANDOMSHASUM=$(echo ${RANDOMSHASUM% *})
    echo "SHA256 SUM: "$RANDOMSHASUM

    # Make sure swarm is running / `ps x | grep [s]warm | awk '{print $1}'`
    local SWARMPID=$(ps x | grep [s]warm | grep -v kubectl | grep -v echo | grep -v snarl | awk '{print $1}')

    if [[ -z ${SWARMPID} ]]; then
        # Delete local storage (This can mess up syncing!!)
        eval $RM_CMD &
        # Run Swarm node
        eval $START_SWARM_CMD &
        sleep 20
    fi

    # Ensure that all nodes are online before uploading
    fail_nodes_random 0 $TOTALPODS

    # Disable Snarl upload
    # local RES=$(eval $SNARLPATH" upload "$RANDOMPATH" "$SNARLDBPATH" --snarldbpath=\$HOME/snarlDBPath/ "$IPCPATH" --numPeers=40 --verbose=true")
    # echo ${RES}

    # Vanilla Swarm upload
    echo $SWARM_UPLOAD_CMD$RANDOMPATH
    eval $SWARM_UPLOAD_CMD$RANDOMPATH #&>/dev/null
    sleep 10
    
    # Simulation to find the content hash
    local RES=$(eval $SNARLPATH" upload "$RANDOMPATH" "$SNARLDBPATH" --snarldbpath=\$HOME/snarlDBPath/ "$IPCPATH" \
    --numPeers=40 --verbose=true --simulate=true")
    echo "SIMULATED: "${RES}
    local FILEPATHS=$RANDOMPATH
    HASHIDS="0x"$(echo ${RES#*Content hash: })

    # Entangle the file
    entangle_file "$RANDOMPATH"
    
    # Upload entangled files
    for f in $ENTANGLEDIR/*
    do
        echo "Uploading $f"
        sleep 15

        # Disable Snarl upload
        # local RES=$(eval $SNARLPATH" upload "$f" "$SNARLDBPATH" --chunkdbpath=\$HOME/snarlDBPath/ "$IPCPATH" --numPeers=40 --verbose=true")
        # echo ${RES}

        # Vanilla Swarm upload
        echo $SWARM_UPLOAD_CMD$f
        eval $SWARM_UPLOAD_CMD$f #&>/dev/null
        sleep 10

        # Get the expected content hash.
        local RES=$(eval $SNARLPATH" upload "$f" "$SNARLDBPATH" --snarldbpath=\$HOME/snarlDBPath/ "$IPCPATH" \
        --numPeers=40 --verbose=true --simulate=true")
        echo "SIMULATED: "${RES}
        local FILEPATHS=$FILEPATHS","$f
        HASHIDS=$HASHIDS",0x"$(echo ${RES#*Content hash: })
    done

    # Sync this upload for 1 minute (Might be too long, or unnecessary)
    sleep 60

    # Generate listfiles.
    generatechunklist ${FILEPATHS} ${HASHIDS}
    FILES="chunks_"$(echo ${HASHIDS} | sed "s/,/,chunks_/g")

    # Verify persistence

    ## 11.08.20 - Enable Verification - Note that ${RANDOMSHASUM} is simply a file name here.
    local CMD=$SWARMTOOLSPATH" listchunks "$TOTALPODS" --verboselistpods=true --filters="$FILES" --verifypersistence="$FILES" > "$RANDOMSHASUM
    echo $CMD
    eval $CMD
    echo "Verifying persistence ..."
    local NONPER=$(cat ${RANDOMSHASUM} | grep "\---.*Verifying" | grep -v "\[[0]\]" | awk -F 'chunks_' '{print $2}' | cut -c1-66)
    echo $NONPER
    local LENNONPER=$(echo $NONPER | grep -o "0x" | wc -l)
    local RETRIES=5
    local SNARLUPLOAD=0


    
    while [[ ${LENNONPER} -gt 0 && ${RETRIES} -gt 0 ]]; do  
            echo "Persistence not verified ... Attempting to upload again!!"

            # Terminate Swarm process.
            local SWARMPID=$(ps x | grep [s]warm | grep -v kubectl | grep -v echo | grep -v snarl | awk '{print $1}')
            kill $SWARMPID 
            echo "Terminating swarm. PID: "$SWARMPID 
            sleep 15

            # Clear storage
            echo "Clear storage"
            eval $RM_CMD
            rm -rf $HOME/snarlDBPath
            sleep 15

            if [[ $SNARLUPLOAD -eq 1 ]]; then
                local SNARLUPLOAD=0
                echo "ATTEMPTING SNARL UPLOAD"
                # Start swarm
                echo "Start swarm"
                eval $START_SWARM_CMD &
                sleep 20

                for root in ${NONPER}
                do
                    echo ${HASHIDS}
                    echo ${root}
                    local FILE=$(echo ${HASHIDS} | awk -F "${root}" '{print $1}' | grep -o "," | wc -l)
                    if [[ $FILE == 0 ]]; then
                        local REUPLOAD=$RANDOMPATH
                    else
                        local REUPLOAD=$ENTANGLEDIR/$((${FILE} - 1))
                    fi

                    echo $SNARLPATH" upload "$REUPLOAD" "$SNARLDBPATH" --chunkdbpath=\$HOME/snarlDBPath/ "$IPCPATH" --numPeers=40 --verbose=true"
                    local RES=$(eval $SNARLPATH" upload "$REUPLOAD" "$SNARLDBPATH" --chunkdbpath=\$HOME/snarlDBPath/ "$IPCPATH" --numPeers=40 --verbose=true")
                    echo ${RES}
                done
            else
                #local SNARLUPLOAD=1 Always attempt Vanilla
                echo "ATTEMPTING VANILLA SWARM UPLOAD"

                for root in ${NONPER}
                do
                    echo ${HASHIDS}
                    echo ${root}
                    local FILE=$(echo ${HASHIDS} | awk -F "${root}" '{print $1}' | grep -o "," | wc -l)
                    if [[ $FILE == 0 ]]; then
                        local REUPLOAD=$RANDOMPATH
                    else
                        local REUPLOAD=$ENTANGLEDIR/$((${FILE} - 1))
                    fi

                    echo $SWARM_UPLOAD_CMD$REUPLOAD
                    eval $SWARM_UPLOAD_CMD$REUPLOAD &>/dev/null
                    sleep 10
                done
            fi
            
            # Verify persistence
            sleep 20
            local CMD=$SWARMTOOLSPATH" listchunks "$TOTALPODS" --verboselistpods=true --filters="$FILES" --verifypersistence="$FILES" > "$RANDOMSHASUM"_"$RETRIES
            echo $CMD
            eval $CMD

            local NONPER=$(cat ${RANDOMSHASUM}_${RETRIES} | grep "\---.*Verifying" | grep -v "\[[0]\]" | awk -F 'chunks_' '{print $2}' | cut -c1-66)
            echo $NONPER
            LENNONPER=$(echo $NONPER | grep -o "0x" | wc -l)
            let RETRIES=RETRIES-1
            echo "LEN NONPER " ${LENNONPER}
    done

    if [[ ${LENNONPER} -gt 0 ]]; then
        echo "UPLOAD NOT SUCCESS!"
        HASHIDS="INVALID HASH IDS"

        # Clean up generated files. (At least we can do this if it doesnt succeed...)
        eval rm $RANDOMPATH
        eval rm -rf $ENTANGLEDIR

    else
        echo "UPLOAD SUCCESS!"
        HASHIDS=$(printf '%x' ${1})","$HASHIDS 
    fi

    echo $HASHIDS


    if [[ ${3} == "true" ]]; then
        # Terminate Swarm process.
        local SWARMPID=$(ps x | grep [s]warm | grep -v kubectl | grep -v echo | grep -v snarl | awk '{print $1}')
        echo "SWARM PID (Kill after upload): "$SWARMPID
        kill $SWARMPID 
    fi
}

# Param1: Size, Param2: Name, Param3: Verbose
function generate_randomfile {
    head -c ${1} </dev/urandom >${2}
    if [[ ${3} == "true" ]]; then
        echo "Generated a random file. Name: ${2}, Size: ${1}"
    fi
}


# Param1: File path
function entangle_file {
    if [[ -z ${1} ]]; then
       echo "Please specify file path."
       exit 1
    fi

    local COMMAND="entangle"
    local CMD=$SNARLPATH" "$COMMAND" "${1}" --doupload=false --listchunks=false"

    #echo $CMD
    local OUTPUT=$(eval ${CMD})

    ENTANGLEDIR="/"$(echo "${OUTPUT#*/}")
}


# Param1: File or root hash
# Param 2 will be the hash IDS, if file is given in parameter 1
function generatechunklist {
    local CMD=$SNARLPATH" listchunks --numPeers=1 "$CHUNKDBPATH" "$SNARLDBPATH" "$IPCPATH" "
    
    local HASHARR=""
    IFS=","; read -ra HASHARR <<< ${2}; IFS=';'
    local CNTR=0
    echo ${1} | tr "," "\n" | while read word; do 
        #echo $CMD${word}
        eval $CMD${word} > "chunks_"${HASHARR[${CNTR}]}
        echo "Stored all chunks for "${HASHARR[${CNTR}]}" in chunks_"${HASHARR[${CNTR}]}
        let CNTR=CNTR+1
    done
}