# Contributing

## Proto Generation

This repo uses protobuf to define the interfaces between several services. To help with this, this
repo relies on [buf](https://buf.build). If you modify the protos you can regenerate them using:

```shell
make proto-gen
```

## Helpful commands

```shell
# See the running containers
docker ps

# You can view the logs from a running container via Docker UI or:
docker logs beacond
docker logs celestia-network-bridge
docker logs celestia-network-validator
docker logs simapp-validator
docker logs reth
```
