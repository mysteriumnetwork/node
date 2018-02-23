#!/usr/bin/env bash

docker-compose up -d
docker cp integration_tests/data/ mysterium-client:/usr/lib/integration_tests
docker-compose exec client /usr/lib/integration_tests/test.sh
