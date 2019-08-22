package main

import (
	"afe/config"
	"flag"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// A HealthChecker determines whether the service is healthy, and returns
// a non-nil error if it is not. It has access to the proxy configuration
// so that it can health determinations based on, e.g., the health of the
// configured backends.
type HealthChecker func(*Proxy) error

// A Proxy implements the http.Handler interface and routes requests
// to backends in its configuration.
type Proxy struct {
	config config.ProxyConfig
	// reverseProxy maps a service domain to the ReverseProxy for that service
	reverseProxy map[string]*httputil.ReverseProxy
	// healthChecker determines whether the service is healthy or not
	healthChecker HealthChecker
}

var configPath = flag.String("config", "config.yaml", "full path to config file")

var rpcDurations = prometheus.NewSummaryVec(
	prometheus.SummaryOpts{
		Name:       "proxy_backend_duration_ms",
		Help:       "Proxy latency distributions for backend requests.",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	},
	[]string{"service"},
)

func init() {
	prometheus.MustRegister(rpcDurations)
}

func main() {
	proxy := Proxy{
		healthChecker: okHealthCheck,
	}

	if err := config.ParseConfigFromFile(*configPath, &proxy.config); err != nil {
		log.Fatal(err)
	}

	log.Printf("config: %+v", proxy.config)

	proxy.reverseProxy = make(map[string]*httputil.ReverseProxy)
	for _, service := range proxy.config.Proxy.Services {
		proxy.reverseProxy[service.Domain] = NewRandomBackendReverseProxy(service.Hosts)
	}

	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/", proxy)
	log.Fatal(http.ListenAndServe(proxy.config.Listen.String(), nil))
}

// ServeHTTP implements the generic proxy.
//
// If this was a real application requests would be proxied based on
// the domain of the target of the incoming request. Since I can't
// trivially configure that, look for the first 's' parameter in the
// URL query string and select the service and backends based on that.
//
// Handles health checks by looking for a "health-check" header. If
// present then the request is not proxied, and an indication of the
// server's health is returned.
func (proxy Proxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	isHealthCheck := req.Header.Get("health-check")
	if isHealthCheck != "" {
		if err := proxy.healthChecker(&proxy); err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		_, err := io.WriteString(w, "ok")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			log.Fatalf("writing health check response failed: %v", err)
		}
		return
	}

	q := req.URL.Query()
	service := q.Get("s")

	if service == "" {
		log.Printf("missing 's' parameter in URL %s\n", req.URL)
		http.Error(w, "service not found", http.StatusNotFound)
		return
	}

	backend, ok := proxy.reverseProxy[service]
	if !ok {
		log.Printf("no reverse proxy for s=%s\n", service)
		http.Error(w, "service not found", http.StatusNotFound)
		return
	}

	q.Del("s")
	req.URL.RawQuery = q.Encode()
	log.Printf("routing request for service %s\n", service)

	var stats httpTraceStats
	ctx := WithHTTPTrace(req.Context(), &stats)
	req = req.WithContext(ctx)

	backend.ServeHTTP(w, req)

	stats.Done()
	log.Printf("Stats: Service(%s) %s\n", service, stats.String())
	rpcDurations.WithLabelValues(service).Observe(float64(stats.LatencyTotal / time.Millisecond))
}

// okHealthChecker is a health checker that always returns no
// errors.
func okHealthCheck(proxy *Proxy) error {
	return nil
}

// NewRandomBackendReverseProxy returns a new httputil.ReverseProxy which will
// direct each request to a randomly selected backend.
func NewRandomBackendReverseProxy(backends []config.HostPort) *httputil.ReverseProxy {
	director := func(req *http.Request) {
		i := rand.Intn(len(backends))
		backend := backends[i]
		req.URL.Scheme = "http" // TODO: In real code this would be https
		req.URL.Host = backend.String()
		log.Printf("final URL: %s", req.URL)
	}

	return &httputil.ReverseProxy{Director: director}
}
