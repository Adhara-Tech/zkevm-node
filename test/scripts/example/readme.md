# Polygon Bridging

## Setup

The `zkevm-node` docker image must be built at least once and every time a change is made to the code.
If you haven't build the `zkevm-node` image yet, you must run:

```bash
make build-docker
```

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

## Examples

In `/test/scripts/example/bridge`
```shell
go run main.go
```

## References

[Polygon Unified Bridge](https://docs.polygon.technology/zkEVM/architecture/unified-LxLy/#overall-flow-of-events)