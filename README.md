# Intro

This is an implementation of a small programming challenge to implement a proxy / load balancer for HTTP requests and route them to a randomly chosen backend.

The code consists of:

- `config.yaml`, a configuration file that defines the behaviour of the proxy, including listening ports and the backends to route to.
- `be/*.go`, code for a server that reads the config file and acts as a backend on all the addresses and ports listed as backends in the configuration. This makes it easy to interactively experiment with the balancer.
- `lb/*.go`, code for the balancer.
- `Dockerfile`, constructs a small Docker image to run `lb`.
- `helm-chart/*`, configuration to deploy the docker image to a Kubernetes cluster (tested with `minikube`)

# Deploying and running (locally)

All commands are run from the root of the repository.

## Build the binaries

```shell
go build be/be.go               # Creates binary be(.exe)
go build lb/lb.go lb/trace.go   # Creates binary lb(.exe)
```

## Run the dummy backend

The dummy backend parses the configuration file, logs the representation, and starts listening on the addresses and ports defined in the `services` section (logging them as it does so).

In one shell, run:

```shell
./be
```

This will not detach from the controlling terminal, for ease of killing with `^C`.

## Run the load balancer / proxy

The load balancer parses the configuration file and starts listening on the address and port defined in the `listen` section.

Ordinarily the proxy would inspect the FQDN in the URL of the incoming request and match that against the `domain` in the configuration file to determine the list of backend hosts to proxy to.

To simplify my development environment I've had the proxy honour an `s` parameter in the URL instead. The code that does this is clearly marked. Omitting the `s` parameter, or providing an invalid value returns a 404, with additional information logged.

```shell
./lb
```

This will not detach from the controlling terminal, for ease of killing with `^C`.

## Other flags

Both commands support a `--config` parameter to specify an alternate location for the configuration file. If not given they assume `config.yaml` is in the current working directory.

## Test in the browser

The following assumes you haven't changed the default configuration. Adjust as necessary if you have.

Open `http://localhost:9090/` and `http://localhost:9091` to connect to the `be` binary. You should see `service: my-service, addr: 127.0.0.1:9090` (or `...:9091`) displayed, and the connection logged in the `be` terminal.

Open `http://localhost:8080/?s=my-service.my-company.com`. The `lb` shell will log which of the two backends has been selected to proxy the request to, the `be` shell will show details of the received request, and the browser should show the response from the given backend.

Reload the `:8080` page a few times, and notice that the selected backend changes at random.

# Building and running (Docker, minikube)

> Note: See e.g. https://docs.bitnami.com/kubernetes/get-started-kubernetes/ for minikube/helm/tiller installation doc

> Note: Instructions suitable for hobbyist deployment.

## Update `config.yaml`

- Set the listen address to the empty string to bind correctly
- Update the list of backends as necessary

## Start minikube (if necessary)

```shell
minikube start
```

## Build for Linux

Only necessary if building on a non-Linux host.

```shell
set GOOS=linux
go build -o lb.linux lb/lb.go lb/trace.go
```

## Build the container image

```shell
docker build -t nikclayton/go-afe:0.1.0 .
```

Update the version number as necessary. A version may exist on DockerHub, in which case you can use that.

Make sure it runs, with

```shell
docker run -it nikclayton/go-afe:0.1.0
```

## Install the new build

```shell
helm delete --purge go-afe
helm install -n go-afe ./helm-chart/go-afe
```

## Check for the service's host/port

```shell
minikube service list
```

# Service SLIs

_Note_: These are SLIs just for this service. There are other metrics you want to monitor at e.g. the host level, like memory or disk space consumed, swapping frequency, network traffic and retransmits, but they don't speak to the reliability of the individual service.

For a load balancing proxy I'd want to know at least the following:

## Latencies

Ideally you want the proxy to be _close_ to the client to start handling their request as quickly as possible, and you want the proxy to be _close_ to the backends so that it can proxy the requests to them with the minimum of overhead. If these latencies get too large you can use this to inform capacity planning (e.g., if the app suddenly becomes popular in a different geographical region you can spin up new proxies in that region to receive user traffic faster)

- Time to process each request. This is not the latency of the whole request, rather, track
  - Time from receiving first byte from the client to last byte from the client
  - Time from sending first byte to backend to last byte to backend
  - Time from sending first byte of the response to the client to the last byte to the client
  - Time to process each request (the internal latency added by the service)

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

For example, this proxy returns an error if it receives a request for a service it is not configured for. That might represent an error on the part of the user or it might represent a configuration error. Some percentage of these errors are to be expected, because users will always mis-type URLs. However, if the percentage of these errors crosses a threshold defined in the SLO then it is a cause for concern - perhaps a configuration with an error has been deployed, or a broken link has been published. This is one of the few times where it can be reasonable for someone to receive an alert about a service, even if the problem is with an unrelated service or system.

In addition you would collect metrics from outside the service to verify its reliability -- if the service is unable to accept new incoming connections because it has exhausted a resource the service metrics would not report that, but metrics collected at e.g., the host level could.

## Other metrics

There are other metrics to collect from the proxy that are not part of the SLI directly, but can help inform why the SLI is not being met. For example, you might see periodic increases in the latency of the service, and to help track that down recording metrics about the Go garbage collector behaviour (frequency of collection, amount of memory processed and reclaimed, duration of the collection) would be helpful. However, these aren't typically part of the SLIs, because they're not part of the SLOs.

Tracking requests/queries received per second is another example of this. An SLO to achieve a particular rps/qps rate is not useful, but recording this (and partitioning it by network, user, service, etc, as described above) helps with capacity planning, identifying denial-of-service attacks, problematic clients, and more.

## Implementation

The lb binary records the time taken to:

- Forward the request to a backend
- Wait for the backend to send a response
- Receive the response from a backend

on a per-service basis, and logs the result in a human-readable format.

It also exposes a Prometheus `/metrics` endpoint with the total backend request latency grouped in to percentile buckets. See `lb/trace.go` for the metric recording.

> Note: This means the proxy will not proxy requests for `/metrics`

# Productionisation

Things I considered doing, didn't do because of the time, but would consider to be part of normal production ready code.

- _Daemonise_ the servers (diassociate themselves from the controlling terminal, chdir to `/`, etc)

- Using a full-featured logging library (log levels, logging to different locations, logging stack traces on failures, only logging every N messages, etc)

- Health checking the backends - a mechanism to ensure the proxy notices if a backend becomes slow or non-responsive, and to (temporarily) remove that backend from the backend pool

- Require HTTPS everywhere.

- ACLs on the endpoints. I would block access to `/metrics` earlier in the network, but it's good defense-in-depth practice to block it here too (e.g., require requests come from IPs known to be internal to the organisation)

- Running control-plane endpoints (`/metrics`, health checking) on ports that are different from the main serving port. The specification didn't ask for this, but it's good practice.

- Health checking should be a library for reuse in other servers, and protected by an ACL.

- Explicit resource requirements in the Helm/Kubernetes configuration.

- Hoisting string constants shared between application and test code in to real constants.

- Configuring an Ingress controller in Kubernetes. This was not necessary for experimentation with `minikube` and the `NodePort` configuration.

- The proxy could inject an HTTP header that identifies the request in logs and pass that header to the backends, which would also log it. This makes debugging the servers that an individual request passes through much simpler.

- Making the load balancing strategy "pluggable", so that it can be specified on a per-service basis. Implementing another strategy (e.g., round-robin) was explicitly not requested. This isn't really part of productionisation, and https://en.wikipedia.org/wiki/You_aren%27t_gonna_need_it applies.
