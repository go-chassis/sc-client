#!/bin/sh
set -e


netstat -lntp

cd $GOPATH/src/github.com/ServiceComb/go-sc-client
#Start unit test
for d in $(go list ./... | grep -v /vendor/); do
    echo $d
    echo $GOPATH
    cd $GOPATH/src/$d
    if [ $(ls | grep _test.go | wc -l) -gt 0 ]; then
        go test 
    fi
done


