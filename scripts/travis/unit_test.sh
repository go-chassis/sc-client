#!/bin/sh
set -e

go get -d -u github.com/stretchr/testify/assert

mkdir -p $GOPATH/src/github.com/gorilla
mkdir -p $GOPATH/src/github.com/cenkalti
mkdir -p $GOPATH/src/golang.org/x

cd $GOPATH/src/github.com/ServiceComb
git clone https://github.com/ServiceComb/go-chassis.git
git clone https://github.com/ServiceComb/http-client.git
git clone https://github.com/ServiceComb/paas-lager.git
git clone https://github.com/ServiceComb/auth.git

cd $GOPATH/src/github.com/gorilla
git clone https://github.com/gorilla/websocket.git
cd websocket
git reset --hard 1f512fc3f05332ba7117626cdfb4e07474e58e60

cd $GOPATH/src/github.com/cenkalti
git clone https://github.com/cenkalti/backoff.git
cd backoff
git reset --hard 3db60c813733fce657c114634171689bbf1f8dee

cd $GOPATH/src/golang.org/x
git clone https://github.com/golang/net.git
git clone https://github.com/golang/text.git

netstat -lntp

cd $GOPATH/src/github.com/ServiceComb/go-sc-client
#Start unit test
for d in $(go list ./...); do
    echo $d
    echo $GOPATH
    cd $GOPATH/src/$d
    if [ $(ls | grep _test.go | wc -l) -gt 0 ]; then
        go test 
    fi
done


