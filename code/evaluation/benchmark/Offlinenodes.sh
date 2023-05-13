#!/bin/bash

NODES=`kubectl get pods --all-namespaces | grep -e "swarm-private-[0-9].*1/1.*" | wc -l`
echo "Numnodes $NODES"
if [[ $NODES -le $1 ]]; then
 exit
fi

TERMNODES=`kubectl get pods --all-namespaces | grep -e "swarm-private-[0-9].*1/1.*Terminating.*" | wc -l`
echo "Terminating nodes $TERMNODES"
if [[ $TERMNODES -ge 3 ]]; then
 exit
fi

TODELETE=$(( RANDOM % 10 ))
echo "ToDel $TODELETE"
ONLINE=`kubectl get pods --all-namespaces | sed -n "s/^.*swarm-private-\([0-9]\).*1\/1.*$/\1/p" | tr '\n' ',' | sed '$s/.$//'`
echo "ONLINE ... $ONLINE"
while [[ ! $ONLINE =~ $TODELETE ]]; do
echo "TOD $TODELETE was NOT in $ONLINE"
    kubectl exec -it swarm-private-1 -n swarm -- /root/ContinueStopSwarm.sh -s &>/dev/null &
done
