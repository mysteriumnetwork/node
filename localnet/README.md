### Local development in docker environment


1. **Start localnet docker stack**

```
go run mage.go -v LocalnetUp
```

2. **Build myst binary for Linux**

```
GOOS=linux ./bin/build
```

3. **Run provider**

Connect to container
```
docker exec -it localnet_myst-provider_1 /bin/bash
```
Run provider
```
./localnet/provider.sh
```

4. **Run consumer**

Connect to container
```
docker exec -it localnet_myst-consumer_1 /bin/bash
```

Run consumer daemon

```
./localnet/consumer.sh
```

Run consumer CLI

```
./localnet/cli.sh
```

5. **Stop localnet docker stack**
```
go run mage.go -v LocalnetDown
```
