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
go build lb/lb.go lb/trace.go
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

# Service SLIs

_Note_: These are SLIs just for this service. There are other metrics you want to monitor at e.g. the host level, like memory or disk space consumed, swapping frequency, network traffic and retransmits, but they don't speak to the reliability of the individual service.

For a load balancing proxy I'd want to know at least:

## Latencies

Ideally you want the proxy to be _close_ to the client to start handling their request as quickly as possible, and you want the proxy to be _close_ to the backends so that it can proxy the requests to them with the minimum of overhead. If these latencies get too large you can use this to inform capacity planning (e.g., if the app suddenly becomes popular in a different geographical region you can spin up new proxies in that region to receive user traffic faster)

- Time to process each request. This is not the latency of the whole request, rather, track
  - Time from receiving first byte from the client to last byte from the client
  - Time from sending first byte to backend to last byte to backend
  - Time from sending first byte of the response to the client to the last byte to the client
  - Time to process each request (the internal latnecy added by the service)

All of this information helps to tell you if the proxies are in appropriate locations.

When reporting latencies bucket them by percentile according to the SLO. Useful buckets generally include:

- 50% (median)
- 99%
- 99.9% (and add more nines as needed by the SLO)
- 100% (what is the absolute worst performance customers are seeing?)

In addition, group these based on observed facts about the client. For example, you may want to group latencies by region -- perhaps your overall latency at 99.9% is below the SLO, but the 99.9% latency for customers in, say, Brazil, is below the target latency. This also depends on the wording of the SLO -- in my experience they start quite general (e.g.., 99.9 percentile latency < 10ms) and become more specific over time as the maturity of the SLO model increases.

Similarly, it can be a good idea to partition latencies by customers or individual users, perhaps by username (if the service requires authentication) or by IP address. This can be very problematic -- you don't want to inadvertently log information that can be considered Personally Identifiable Information (PII) due to obvious privacy concerns. However, this can allow you to identify situations where the majority of users are seeing latency within the SLO, but one user, or a subset of users, are seeing latency that is outside the SLO. Again, this depends on how precisely the SLO is written.

As a proxy that supports multiple distinct services (with different backends) these values should be partitioned by service, and probably by individual backend server too. Again, this doesn't directly feed in to an SLI, but is very useful to assist in debugging why latency has suddenly changed, and if the change affects all services the proxy is forwarding for, some subset of them, or some subset of the hosts.

The tradeoff is that as you increase the cardinality of the data being collected the data storage requirements grow and it can place an increased burden on the monitoring and reporting infrastructure.

## Responses

The service should respond reliably. As a proxy the responses from the service form two groups; responses its forwarding from a backend, and responses it's generating itself.

The proxy can not control the responses it receives from a backend, so they should not be part of the SLIs for the proxy (however, they would be part of an SLO that considers the "application service stack" as a whole, and monitoring metrics for the responses from the backends via the proxy is a sensible approach). There are limited situations where the proxy can generate its own error responses, and those should be tracked.

For example, this proxy returns an error if it receives a request for a service it is not configured for. That might represent an error on the part of the user or it might represent a configuration error. Some percentage of these errors are to be expected, because users will always mis-type URLs. However, if the percentage of these errors crosses a threshold defined in the SLO then it is a cause for concern.

In addition you would collect metrics from outside the service to verify its reliability -- if the service is unable to accept new incoming connections because it has exhausted a resource the service metrics would not report that, but metrics collected at e.g., the host level could.

## Other metrics

There are other metrics to collect from the proxy that are not part of the SLI directly, but can help inform why the SLI is not being met. For example, you might see periodic increases in the latency of the service, and to help track that down recording metrics about the Go garbage collector behaviour (frequency of collection, amount of memory processed and reclaimed, duration of the collection) would be helpful. However, these aren't typically part of the SLIs, because they're not part of the SLOs.

Tracking requests/queries received per second is another example of this. An SLO to achieve a particular rps/qps rate is not useful, but recording this (and partitioning it by network, user, service, etc, as described above) helps with capacity planning, identifying denial-of-service attacks, problematic clients, and more.

## Implementation

The lb binary records the network durations of

- Forwarding the request to a backend
- Receiving the response from a backend

on a per-service basis, and logs the result in a human-readable format.

# Requirements

- [x] Implement the proxy with a random-forwarding load balancing policy

- [ ] Provide a helm chart to deploy the proxy

- [x] Define the main SLI that guarantee reliability, performance, and scalability

- [-] Choose and implement _one_ of the SLIs

# Options considered

Things I considered doing, didn't do because of the time, but would consider to be part of normal production ready code.

- Using a non-std logging library (log levels, logging to different locations, stack traces, etc)

- Health checking the backends - a mechanism to ensure the proxy notices if a backend becomes slow or non-responsive, and to (temporarily) remove that backend from the backend pool
