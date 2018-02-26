#!/usr/bin/env bash

docker-compose up -d
result=`curl localhost:4050/connection`

if [ "$result" != '{"status":"NotConnected"}' ]
then
    echo "Unexpected status response: $result"
    exit 1
fi

echo "Tests passed"
exit 0
