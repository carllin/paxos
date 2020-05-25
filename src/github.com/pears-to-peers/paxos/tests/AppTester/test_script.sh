#!/bin/bash

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

#build testing script
go install github.com/pears-to-peers/paxos/tests/AppTester/
if [ $? -ne 0 ]; then
   echo "FAIL: code does not compile"
   exit $?
fi

APP_TEST=$GOPATH/bin/AppTester

#Run Tests
echo Y Y Y| ${APP_TEST} 1





