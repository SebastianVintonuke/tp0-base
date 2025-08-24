#!/bin/bash

message="[CLIENT 0] Message N°0"
server_ip=$(grep 'SERVER_IP' server/config.ini | cut -d'=' -f2 | tr -d ' ')
server_port=$(grep 'SERVER_PORT' server/config.ini | cut -d'=' -f2 | tr -d ' ')

response=$(docker run --rm --network tp0_testing_net alpine:latest sh -c "echo '$message' | nc $server_ip $server_port")

if [ "$response" = "$message" ]; then
    echo "action: test_echo_server | result: success"
    exit 0
else
    echo "action: test_echo_server | result: fail"
    exit 1
fi