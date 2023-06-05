# DistPinger

This repository contains source code for the node of the DistPinger P2P network which is designed to help people around the world to check the availability and stability of websites on the internet.

Unlike in existing solutions where users interact with a single control server that issues website status requests to a set of predefined testing probes around the world, in DistPinger each network member acts both as a client and a probe, and to get a result of the availability test, user requests website status directly from other users. This design increases the resilience of the whole system and makes it possible not to rely on a single information provider while at the same time potentially expanding the set of available testing probes.

To overcome the problem of malicious users in such an open network, DistPinger also implements a reputation system that mitigates the impact of harmful activity in the system.

## Build
To build this project you must setup required [prerequisites](https://grpc.io/docs/languages/go/quickstart/#prerequisites) for the Google protocol buffer compiler to generate Go code and run the following command from the root of the repository to generate gRPC bindings:
```bash
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative grpc/*.proto
``` 

After that the repository can be built as a normal Golang module. 

## Usage
To start a node, the following flags must be specified:

- `address`, external address on which this node will be available for other users;
- `port`, port on which the probe server will be running;

One of the following type of arguments should also be used to connect to a network:

- flag `ref`, an ID of a trustworthy DistPinger member that would be used on start as a source of information about other nodes and an entry point to the network;
- non-flag arguments, IDs of trustworthy DistPinger users who will be known and trusted by node from the start;

If there is an existing network, users will likely prefer the `ref` method as it allows them to easily join the system knowing only an ID of one other user. However the second method is required to create a "reference group" in a new DistPinger network.

> After connecting to a network you just need to enter the address of a web service you want to check to perform an availability test.
