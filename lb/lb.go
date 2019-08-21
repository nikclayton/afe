package main

import (
	"afe/config"
	"flag"
	"log"
	"math/rand"
	"net/http"
	"net/http/httputil"
)

type Proxy struct {
	config config.ProxyConfig
	// reverseProxy maps a service domain to the ReverseProxy for that service
	reverseProxy map[string]*httputil.ReverseProxy
}

var configPath = flag.String("config", "config.yaml", "full path to config file")

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

	http.HandleFunc("/", proxy.handler)
	log.Fatal(http.ListenAndServe(proxy.config.Listen.String(), nil))
}

// handler implements the generic proxy.
//
// If this was a real application requests would be proxied based on
// the domain of the target of the incoming request. Since I can't
// trivially configure that, look for the first 's' parameter in the
// URL query string and select the service and backends based on that.
func (proxy Proxy) handler(w http.ResponseWriter, req *http.Request) {
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
