# Steps to run/develop on the environment locally

## Overview

This documentation will help you running the following components:

- zkEVM Node Databases
- Explorer Databases
- L1 Network
- Prover
- zkEVM Node components
- Explorers

## Requirements

The current version of the environment requires `go`, `docker` and `docker-compose` to be previously installed, check the links below to understand how to install them:

- <https://go.dev/doc/install>
- <https://www.docker.com/get-started>
- <https://docs.docker.com/compose/install/>

The `zkevm-node` docker image must be built at least once and every time a change is made to the code.
If you haven't build the `zkevm-node` image yet, you must run:

```bash
make build-docker
```

## Controlling the environment

> All the data is stored inside of each docker container, this means once you remove the container, the data will be lost.

To run the environment:

The `test/` directory contains scripts and files for developing and debugging.

```bash
cd test/
```

Then:

```bash
make run
```

To stop the environment:

```bash
make stop
```

To restart the environment:

```bash
make restart
```

## Scripts 

```shell
go run test/scripts/bridging/deposit/main.go
```