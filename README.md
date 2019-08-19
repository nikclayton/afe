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

## Build the load balancer / proxy

The load balancer parses the configuration file and starts listening on the address and port defined in the `listen` section.

In another shell,

```shell
go build lb/lb.go
```

## Run the load balancer / proxy

Ordinarily the proxy would inspect the FQDN in the URL of the incoming request and match that against the `domain` in the configuration file to determine the list of backend hosts to proxy to.

To simplify my development environment I've had the proxy honour an `s` parameter in the URL instead. The code that does this is clearly marked. Omitting the `s` parameter, or providing an invalid value returns a 404, with additional information logged.

```shell
./lb
```

## Other flags

Both commands support a `--config` parameter to specify an alternate location for the configuration file.

## Test in the browser

The following assumes you haven't changed the default configuration. Adjust as necessasry.

Visit `http://localhost:8080/?s=my-service.my-company.com`. The `lb` shell will log which of the two backends has been selected to proxy the request to, the `be` shell will show details of the received request, and the browser should show which of the two addresses was selected.

# Requirements

- [x] Implement the proxy with a random-forwarding load balancing policy

- [ ] Provide a helm chart to deploy the proxy

- [ ] Define the main SLI that guarantee reliability, performance, and scalability

- [ ] Choose and implement _one_ of the SLIs

# Options considered

Things I considered doing, didn't do because of the time, but would consider to be part of normal production ready code.

- Using a non-std logging library (log levels, logging to different locations, stack traces, etc)
