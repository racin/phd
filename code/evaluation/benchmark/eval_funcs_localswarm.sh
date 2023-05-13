# Param1: Numpeers, Param2: UpdateWant
function connect_to_peers {
    local initPeers=$(/usr/bin/geth --exec "admin.peers.length" attach $HOME/.ethereum/bzzd.ipc)

    # If Param2 is true, we will define this variable.
    if [[ ${2} == "true" ]]; then
        wantPeers=${1}
        MAXITER=${1}
    fi
    
    if [[ ${initPeers} == ${wantPeers} ]]; then
        echo "Already connected to ${wantPeers} peers. Returning."
        return
    fi

    local ITERATIONS=0

    while [[ $ITERATIONS -lt ${MAXITER} ]]; do
        if [[ $CONNTOPEERSFINISH == "true" ]]; then
            echo "We are already connected to enough peers (${wantPeers})."
            break
        fi
        # If the current iteration is in failed nodes, do not attempt to connect.
        if [[ " ${FAILEDNODES} " =~ " ${ITERATIONS} " ]]; then
            echo "Peer swarm-private-${ITERATIONS} is failed. Will not connect."
            if [[ ${2} == "true" ]]; then
                let wantPeers=wantPeers-1
            fi
        else
            connect_to_swarm_peer "swarm-private-${ITERATIONS}"
        fi

        if [[ $(($ITERATIONS%100)) == 0 ]]; then
            echo "Current iteration: $ITERATIONS. Finish status: $CONNTOPEERSFINISH"
        fi
        let ITERATIONS=ITERATIONS+1
    done

    local endPeers=$(/usr/bin/geth --exec "admin.peers.length" attach $HOME/.ethereum/bzzd.ipc)
    echo "Started with ${initPeers} peers, wanted ${wantPeers} peers. Is now connected to ${endPeers} peers."

    if [[ ${endPeers} -lt ${wantPeers} ]]; then
        echo "Wanted to be connected to ${wantPeers}. Retrying..."
        connect_to_peers ${wantPeers} "false"
    fi
}

# Param1: Peer id
function connect_to_swarm_peer {
    # Query every pod for its enode address
    local enode=$(eval "kubectl exec ${1} -n swarm -- /usr/local/bin/geth \"--exec\" \"admin.nodeInfo.enode\" \"attach\" \"/root/.ethereum/bzzd.ipc\"")
    #echo $ITERATIONS" --- "$enode
    if [[ ! $enode =~ "enode" ]]; then
        # We were not able to attach to the geth console. Try to restart this node.
        #echo "Trying to restart node ${1}."

        kubectl delete pod ${1} --grace-period=30 -n swarm
        echo "Trying to connect to node ${1} again in 40 seconds"

        # Decrement want peers
        #let wantPeers=wantPeers-1
        #echo "Could not connect to ${1}. Decrementing wantPeers to ${wantPeers}."

        sleep 40
        connect_to_swarm_peer ${1}
    fi

    # Connect local swarm node to that peer (pod)
    /usr/bin/geth --exec "admin.addPeer(${enode})" attach $HOME/.ethereum/bzzd.ipc &>/dev/null
}