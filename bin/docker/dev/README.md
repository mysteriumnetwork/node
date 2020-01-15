### Local development in docker

While developing locally you may need to have more complex setup like provider running behind router etc. For that e2e nat tests infrastructure
could be reused which already setups all needed parts including local broker, api, ganache, router etc.

#### Guide

1. Changes in `e2e/traversal/docker-compose.yml`

    1.1 Point myst-consumer and myst-provider to use `bin/docker/dev/Dockerfile` image.
    
    1.2 Add volume for path `- ../..:/node` for both myst-consumer and myst-provider.
    
2. Run docker-compose stack

```
go run mage.go -v NATDevUp
```

3. Connect to provider container

```
docker exec -it $(docker ps -aqf "name=node_e2e_nat_test_myst-provider_1") /bin/bash
```

4. Connect to consumer container
```
docker exec -it $(docker ps -aqf "name=node_e2e_nat_test_myst-consumer_1") /bin/bash
```

5. Use helper scripts to build source code and run provider or consumer.
```
# Build and run provider
./bin/build && ./bin/docker/dev/provider.sh

# Build and run consumser daemon
./bin/build && ./bin/docker/dev/consumser.sh

# Run consumser CLI
./bin/docker/dev/cli.sh
```
