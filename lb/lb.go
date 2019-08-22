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

type Proxy struct {
	config config.ProxyConfig
	// reverseProxy maps a service domain to the ReverseProxy for that service
	reverseProxy map[string]*httputil.ReverseProxy
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
	proxy := Proxy{}

	if err := config.ParseConfigFromFile(*configPath, &proxy.config); err != nil {
		log.Fatal(err)
	}

	log.Printf("config: %+v", proxy.config)

	proxy.reverseProxy = make(map[string]*httputil.ReverseProxy)
	for _, service := range proxy.config.Proxy.Services {
		proxy.reverseProxy[service.Domain] = NewRandomBackendReverseProxy(service.Hosts)
	}

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", proxy.handler)
	log.Fatal(http.ListenAndServe(proxy.config.Listen.String(), nil))
}

// handler implements the generic proxy.
//
// If this was a real application requests would be proxied based on
// the domain of the target of the incoming request. Since I can't
// trivially configure that, look for the first 's' parameter in the
// URL query string and select the service and backends based on that.
//
// Handles health checks by looking for a "health-check" header. If
// present then the request is not proxied, and an indication of the
// server's health is returned.
func (proxy Proxy) handler(w http.ResponseWriter, req *http.Request) {
	health := req.Header.Get("health-check")
	if health != "" {
		_, err := io.WriteString(w, "ok")
		if err != nil {
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
