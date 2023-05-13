# Param1: TOFAIL, Param2: TOTALNODES 
function fail_nodes_random {
    local NUMNODES=${1}
    local TOTALNODES=${2}
    local FAILED=0
    local TOFAILCNT=0
    local ITERATIONS=0
    local OLDFAILED=${FAILEDNODES[@]}
    local OLDONLINE=${ONLINENODES[@]}
    local NEWFAILED
    local NEWONLINE

    # Randomly select which nodes(pods) we want to fail.
    while [[ $TOFAILCNT -lt $NUMNODES ]]; do
        local TOFAIL=$(( RANDOM % ${TOTALNODES} ))
        if [[ ! " ${NEWFAILED[@]} " =~ " ${TOFAIL} " ]]; then
            let TOFAILCNT=TOFAILCNT+1
            NEWFAILED+=($TOFAIL)
            local NEWFAILEDSTR=${NEWFAILED[@]}
        fi
    done

    # Restore the rest
    while [[ $ITERATIONS -lt $TOTALNODES ]]; do
       
        if [[ " ${NEWFAILEDSTR[@]} " =~ " ${ITERATIONS} " ]]; then
            # Node is already failed - no action required.
            if [[ ! " ${OLDFAILED[@]} " =~ " ${ITERATIONS} " ]]; then
                # STOP
                eval "kubectl exec swarm-private-"${ITERATIONS}" -n swarm -- /bin/sh \"/root/ContinueStopSwarm.sh\" \"-s\"" &
                #echo "kubectl exec swarm-private-"${ITERATIONS}" -n swarm -- /bin/sh \"/root/ContinueStopSwarm.sh\" \"-s\""
                #echo "STOPPED "${ITERATIONS}
            fi
        else
            if [[ ! " ${OLDONLINE[@]} " =~ " ${ITERATIONS} " ]]; then
                # CONTINUE
                eval "kubectl exec swarm-private-"${ITERATIONS}" -n swarm -- /bin/sh \"/root/ContinueStopSwarm.sh\" \"-c\"" &
                #echo "CONTINUE "${ITERATIONS}
            fi
            NEWONLINE+=($ITERATIONS)
        fi
        let ITERATIONS=ITERATIONS+1
    done
    FAILEDNODES=${NEWFAILED[@]}
    ONLINENODES=${NEWONLINE[@]}
    echo $FAILEDNODES > FAILEDNODES
    echo $ONLINENODES > ONLINENODES

    # echo "FAILED: "${FAILEDNODES}
    # echo "ONLINE: "${ONLINENODES}
}

# Param1: TOTALNODES, Param2: TOFAIL
function fail_nodes_deterministic {
    local TOTALNODES=${1}
    local FAILED=0
    local TOFAILCNT=0
    local ITERATIONS=0
    local OLDFAILED=${FAILEDNODES[@]}
    local OLDONLINE=${ONLINENODES[@]}
    local NEWFAILED
    local NEWONLINE

    # Select nodes that we will fail
    IFS=','
    read -ra NEWFAILED <<< ${2}
    local NEWFAILEDSTR=${NEWFAILED[@]}

    # Restore the rest
    while [[ $ITERATIONS -lt $TOTALNODES ]]; do
       
        if [[ " ${NEWFAILEDSTR[@]} " =~ " ${ITERATIONS} " ]]; then
            # Node is already failed - no action required.
            if [[ ! " ${OLDFAILED[@]} " =~ " ${ITERATIONS} " ]]; then
                # STOP
                eval "kubectl exec swarm-private-"${ITERATIONS}" -n swarm -- /bin/sh \"/root/ContinueStopSwarm.sh\" \"-s\"" &
                #echo "kubectl exec swarm-private-"${ITERATIONS}" -n swarm -- /bin/sh \"/root/ContinueStopSwarm.sh\" \"-s\""
            fi
        else
            if [[ ! " ${OLDONLINE[@]} " =~ " ${ITERATIONS} " ]]; then
                # CONTINUE
                eval "kubectl exec swarm-private-"${ITERATIONS}" -n swarm -- /bin/sh \"/root/ContinueStopSwarm.sh\" \"-c\"" &
            fi
            NEWONLINE+=($ITERATIONS)
        fi
        let ITERATIONS=ITERATIONS+1
    done
    FAILEDNODES=${NEWFAILED[@]}
    ONLINENODES=${NEWONLINE[@]}
    echo $FAILEDNODES > FAILEDNODES
    echo $ONLINENODES > ONLINENODES

    echo "FAILED: "${FAILEDNODES}
    echo "ONLINE: "${ONLINENODES}
}

# Terminates all pods. Keeps running until they are all terminated.
# Param1: Number of checks (Checks every 10th second)
TERMINATEPODSRESULT=""
function terminatePods {
    helm_change_running_status "false"
    sleep 20
    helm_change_running_status "restart"
    local TESTCOUNTER=0
    local TESTLIMIT=${1}

    while [[ $TESTCOUNTER -lt $TESTLIMIT ]]; do
        local RUN=$(kubectl get pods -n swarm 2>&1)
        local DELFILEDESC=$(sudo pdsh -w bbchain[3-30] lsof 2>/dev/null | grep -e ".*swarm.*(deleted)" | wc -l)
        let TESTCOUNTER=TESTCOUNTER+1
        if [[ $RUN =~ "No resources found in swarm namespace." && $DELFILEDESC == 0 ]]; then
            TERMINATEPODSRESULT="ALL_TERMINATED"
            break
        elif [[ $TESTCOUNTER -eq $TESTLIMIT ]]; then
            TERMINATEPODSRESULT="ERROR"
            break
        fi
        local RUNCOUNT=$(echo $RUN | grep -o "swarm-private" | wc -l)
        echo $RUNCOUNT" pods still running. $DELFILEDESC file descriptors reporting deleted."
        sleep 10
    done
    echo $TERMINATEPODSRESULT
}

# 1) Terminates all pods. 2) Restores storage from backup. 3) Starts all pods again.
# Param1: Replication
function restartAndRestorePodsFromBackup {
    # Try to restart for 30 * 10 = 300 second (5 minutes).
    terminatePods 30
    
    if [[ $TERMINATEPODSRESULT != "ERROR" ]]; then
        quick_restore_pv_directories ${1}
        if [[ $QUICKRESTORERESULT != "ERROR" ]]; then
            helm_change_running_status "true"

            # After restarting - Make sure we update the IPs so that we can connect again.
            updateIPsFromPods ${1}

            # Delete ONLINENODES and FAILEDNODES lists
            rm -f ONLINENODES
            rm -f FAILEDNODES

            ONLINENODES=""
            FAILEDNODES=""
        else
            echo "BACKUP WAS NOT RESTORED CORRECTLY."
        fi
    else
        echo "PODS DID NOT TERMINATE CORRECTLY."
    fi
}


function helm_change_sync {
    if [[ ${1} == "true" ]]; then
        ssh bbchain1 "sed -i -e 's/syncing: false/syncing: true/g' ${BBCHAIN1HOME}helm-snarl/swarm-chart/values.yaml"
    else
        ssh bbchain1 "sed -i -e 's/syncing: true/syncing: false/g' ${BBCHAIN1HOME}helm-snarl/swarm-chart/values.yaml"
    fi
}

function helm_change_running_status {
    if [[ ${1} == "true" ]]; then
        ssh bbchain1 "/opt/helm/linux-amd64/helm install -f ${BBCHAIN1HOME}helm-snarl/swarm-chart/values.yaml --wait --timeout 15m0s --atomic --namespace swarm --create-namespace swarm-private ${BBCHAIN1HOME}helm-snarl/swarm-chart"
    elif [ ${1} == "restart" ]; then
        kubectl delete pods --grace-period=30 -n swarm --all &>/dev/null # TODO: Filter output to only show pod 999 and boot
        echo "Pods deleted."
    else
        ssh bbchain1 "/opt/helm/linux-amd64/helm uninstall swarm-private --namespace swarm"
    fi
}
