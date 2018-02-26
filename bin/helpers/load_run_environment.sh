#!/bin/bash

if [ -f .env ]; then
   source .env

   [ -n "$MYSTERIUM_API_URL" ] && DISCOVERY_OPTION="--discovery-address=$MYSTERIUM_API_URL"
   [ -n "$NATS_SERVER_IP" ] && BROKER_OPTION="--broker-address=$NATS_SERVER_IP"
fi
