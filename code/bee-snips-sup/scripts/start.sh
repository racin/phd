#!/bin/sh

myhostname=$(hostname)
id=$(hostname | cut -d - -f3)

if [ "$id" -eq "0" ]; then
	exec bee start --config=.bee.yaml --bootnode-mode=true --hostname="${myhostname}" --welcome-message="Welcome to ${myhostname}!"
else
	exec bee start --config=.bee.yaml --hostname="${myhostname}" --welcome-message="Welcome to ${myhostname}!"
fi

