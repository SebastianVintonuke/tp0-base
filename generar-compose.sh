#!/bin/bash

output_file_name=$1
clients_number=$2

if [ $# -ne 2 ] || ! [[ "$clients_number" =~ ^[0-9]+$ ]]; then
    echo "$0 <Nombre del archivo de salida> <Cantidad de clientes>"
    exit 1
fi

echo "Nombre del archivo de salida: $1"
echo "Cantidad de clientes: $2"

echo "name: tp0
services:
  server:
    container_name: server
    image: server:latest
    entrypoint: python3 /main.py
    environment:
      - PYTHONUNBUFFERED=1
    networks:
      - testing_net
    volumes:
      - ./server/config.ini:/config.ini:ro
" > "$output_file_name"

for ((i=1; i<=clients_number; i++)); do
  echo "  client$i:
    container_name: client$i
    image: client:latest
    entrypoint: /client
    environment:
      - CLI_ID=$i
      - CLI_FILE=/agency-$i.csv
    networks:
      - testing_net
    depends_on:
      - server
    volumes:
      - ./client/config.yaml:/config.yaml:ro
      - ./.data/agency-${i}.csv:/agency-${i}.csv:ro
" >> "$output_file_name"
done

echo "networks:
  testing_net:
    ipam:
      driver: default
      config:
        - subnet: 172.25.125.0/24" >> "$output_file_name"