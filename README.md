# Building and running

All commands are run from within the `afe` directory.

## Build the dummy backend

The dummy backend parses the configuration file and starts listening on the addresses and ports defined in the `services` section.

```shell
go build be/be.go
```

## Run the dummy backend

```shell
./be
```

It logs a representation of the parsed configuration and all of the address/port pairs it is listening on.

# Requirements

- [ ] Implement the proxy with a random-forwarding load balancing policy

- [ ] Provide a helm chart to deploy the proxy

- [ ] Define the main SLI that guarantee reliability, performance, and scalability

- [ ] Choose and implement _one_ of the SLIs

# Options considered

Things I considered doing, didn't do because of the time, but would consider to be part of normal production ready code.

- Using a non-std logging library (log levels, logging to different locations, stack traces, etc)
