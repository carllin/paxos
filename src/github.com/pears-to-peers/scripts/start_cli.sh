#!/bin/bash
# These are the steps to start the game
# Start a cli
# invoke the scripts that start players and contact the central server via AddPlayer rpc call
# once the central server replies that all players have joined, the CLIs for the players can go ahead with the leader election


if [ -z $GOPATH ]; then
    echo "FAIL: GOPATH environment variable is not set"
    exit 1
fi

if [ -n "$(go version | grep 'darwin/amd64')" ]; then    
    GOOS="darwin_amd64"
elif [ -n "$(go version | grep 'linux/amd64')" ]; then
    GOOS="linux_amd64"
else
    echo "FAIL: only 64-bit Mac OS X and Linux operating systems are supported"
    exit 1
fi

# Build the central server binary
# Exit immediately if there was a compile-time error.
go install github.com/pears-to-peers/cli
go install github.com/pears-to-peers/runners/cli
if [ $? -ne 0 ]; then
   echo "FAIL: code does not compile"
   exit $?
fi

# Pick random ports between [10000, 20000).
if test -z "$PORT"; then
	CLI_PORT=$(((RANDOM % 10000) + 10000))
	CLI=$GOPATH/bin/cli 

	${CLI} -serverport=9999 -port=${CLI_PORT} -quiescing=$QUIESCING -cluster=$CLUSTER
	CLI_PID=$!

else
	CLI_PORT=$PORT
	CLI=$GOPATH/bin/cli 

	${CLI} -serverport=9999 -port=${CLI_PORT} -quiescing=$QUIESCING -cluster=$CLUSTER
	CLI_PID=$!

fi



# Kill the server.
#kill -9 ${CENTRAL_SERVER_PID}
#wait ${CENTRAL_SERVER_PID} 2> /dev/null
