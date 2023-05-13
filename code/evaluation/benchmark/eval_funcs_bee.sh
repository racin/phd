function bee_delete_statestore {
    local ITERATIONS=0
    checkArgs ${1} "number_of_pods"
    while [[ $ITERATIONS -lt ${1} ]]; do
        bbnode=$((${ITERATIONS} % 28 + 3))
        #echo "Deleting all PVs in bbchain${1}"
        echo "ssh -t bbchain${bbnode} \"rm -rf ${PV_ROOT}/pv-${ITERATIONS}/statestore/*\""
        ssh -t bbchain${bbnode} "rm -rf ${PV_ROOT}/pv-${ITERATIONS}/statestore/*"
        let ITERATIONS=ITERATIONS+1
    done
}

function bee_delete_all {
    local ITERATIONS=0
    checkArgs ${1} "number_of_pods"
    while [[ $ITERATIONS -lt ${1} ]]; do
        bbnode=$((${ITERATIONS} % 28 + 3))
        #echo "Deleting all PVs in bbchain${1}"
        echo "ssh -t bbchain${bbnode} \"rm -rf ${PV_ROOT}/pv-${ITERATIONS}/*\""
        ssh -t bbchain${bbnode} "rm -rf ${PV_ROOT}/pv-${ITERATIONS}/*"
        let ITERATIONS=ITERATIONS+1
    done
}

function bee_fund_nodes {
    while [[ $ITERATIONS -lt $TOTALPODS ]]; do
        # Fund with ETH
        kubectl --insecure-skip-tls-verify=true logs pod-snarlbee-${ITERATIONS} -n snarlbee | grep "xDAI network available on " | head -n 1 | awk -F 'available on ' '{print $2}' | sed 's/.$//'

        let ITERATIONS=ITERATIONS+1
    done   
}

function bee_fund_nodes_bzz {
    while [[ $ITERATIONS -lt $TOTALPODS ]]; do
        # Fund with BZZ
        kubectl --insecure-skip-tls-verify=true logs pod-snarlbee-${ITERATIONS} -n snarlbee | grep "BZZ available on " | head -n 1 | awk -F 'BZZ available on ' '{print $2}' | sed 's/.$//'
        let ITERATIONS=ITERATIONS+1
    done   
}

function appendToAddressBook {
    checkArgs ${1} "pod_name"
    checkArgs ${2} "eth_addr"
    checkArgs ${3} "filename"

    echo "${1};0x${2}" >> ${3}
}

function getEthAddress {
    local ITERATIONS=0
    checkArgs ${1} "number_of_pods"
    while [[ $ITERATIONS -lt ${1} ]]; do
        ADDRESS=""

        # Need ETH funding?
        ADDRESS=$(kubectl --insecure-skip-tls-verify=true logs pod-snarlbee-${ITERATIONS} -n snarlbee -c snarlbee | grep "xDAI network available on " | head -n 1 | awk -F '"address"=' '{print $2}' | sed 's/"//g' | sed 's/0x//g')
        
        if [[ -z $ADDRESS  ]]; then
            # Need BZZ funcing?
            ADDRESS=$(kubectl --insecure-skip-tls-verify=true logs pod-snarlbee-${ITERATIONS} -n snarlbee -c snarlbee | grep "BZZ available on " | head -n 1 | awk -F '"address"=' '{print $2}' | sed 's/"//g' | sed 's/0x//g')

            if [[ -z $ADDRESS ]]; then
                # Already running?
                ADDRESS=$(kubectl --insecure-skip-tls-verify=true -n snarlbee -c snarlbee exec pod-snarlbee-${ITERATIONS} -- cat //home/bee/.bee/keys/swarm.key | jq .address | cut -c 2- | sed 's/.$//')

                if [[ -z $ADDRESS ]]; then
                    fatal "Could not ethereum address of pod pod-snarlbee-${ITERATIONS}";
                fi
            fi
        fi

        appendToAddressBook pod-snarlbee-${ITERATIONS} $ADDRESS "podaddresses.csv";
        let ITERATIONS=ITERATIONS+1
    done
}