package main

import (
	"afe/config"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"testing"
)

var goldenConfig = config.ProxyConfig{
	Proxy: config.Proxy{
		Listen: config.HostPort{
			Address: "127.0.0.1",
			Port:    8080,
		},
		Services: []config.Service{{
			Name:   "my-service",
			Domain: "my-service.my-company.com",
			Hosts: []config.HostPort{{
				Address: "127.0.0.1",
				Port:    9090,
			}, {
				Address: "127.0.0.1",
				Port:    9091,
			}},
		}},
	},
}

// TestHealthChecksOK verifies that health checks expected to succeed
// do succeed.
func TestHealthChecksOK(t *testing.T) {
	proxy := Proxy{
		config:        goldenConfig,
		healthChecker: okHealthCheck,
	}

	ts := httptest.NewServer(http.HandlerFunc(proxy.handler))
	defer ts.Close()

	req, _ := http.NewRequest("GET", ts.URL, nil)
	req.Header.Set("health-check", "health-check")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("could not read response body: %+v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("got %d, want 200 as status code", resp.StatusCode)
	}
	if string(result) != "ok" {
		t.Fatalf("got '%s', want 'ok' as response body", result)
	}
}

// TestHealthChecksFail verifies that health checks expected to fail
// do fail.
func TestHealthChecksFail(t *testing.T) {
	checkErr := errors.New("failed health check")
	proxy := Proxy{
		config: goldenConfig,
		healthChecker: func(proxy *Proxy) error {
			return checkErr
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(proxy.handler))
	defer ts.Close()

	req, _ := http.NewRequest("GET", ts.URL, nil)
	req.Header.Set("health-check", "health-check")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("could not read response body: %+v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("got %d, want %d", resp.StatusCode, http.StatusServiceUnavailable)
	}
	if strings.TrimSpace(string(result)) != checkErr.Error() {
		t.Fatalf("got '%s', want '%s' as response body", result, checkErr)
	}
}

// TestMissingSParam verifies that requests without an s= parameter
// generate an error.
func TestMissingSParam(t *testing.T) {
	proxy := Proxy{
		config:        goldenConfig,
		healthChecker: okHealthCheck,
	}

	ts := httptest.NewServer(http.HandlerFunc(proxy.handler))
	defer ts.Close()

	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("could not read response body: %+v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("got %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
	if strings.TrimSpace(string(result)) != "service not found" {
		t.Fatalf("got '%s', want '%s' as response body", result, "service not found")
	}
}

// TestInvalidSParam verifies that requests with an invalid s= parameter
// generate an error.
//
// Note that this is the same code as TestMissingSParam because they generate
// the same results. In a real service I would expect that the server would
// generate different responses for internal requests that include more
// detailed debugging information that would be differentiated in the tests.
func TestInvalidSParam(t *testing.T) {
	proxy := Proxy{
		config:        goldenConfig,
		healthChecker: okHealthCheck,
	}

	ts := httptest.NewServer(http.HandlerFunc(proxy.handler))
	defer ts.Close()

	resp, err := http.Get(fmt.Sprintf("%s/?s=foo", ts.URL))
	if err != nil {
		t.Fatal(err)
	}
	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("could not read response body: %+v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("got %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
	if strings.TrimSpace(string(result)) != "service not found" {
		t.Fatalf("got '%s', want '%s' as response body", result, "service not found")
	}
}

// Note: No need to check to see if health checks with missing s= params
// work, as the parameter is not set in the existing health check code.

// TestProxyFunctionality starts a test server to act as a backend to the
// proxy, then configures the proxy to use it as a backed, connects to
// the proxy and verifies that the expected result is returned.
func TestProxyFunctionality(t *testing.T) {
	proxy := Proxy{
		config:        goldenConfig,
		healthChecker: okHealthCheck,
	}

	// Backend server to proxy for. Start it running, and update the proxy
	// config so it's the only host.
	backendResp := "this is the backend"
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, backendResp)
	}))
	defer backend.Close()

	backendUrl, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatalf("could not parse '%s' as a URL: %+v", backend.URL, err)
	}

	parsedPort, err := strconv.ParseInt(backendUrl.Port(), 10, 0)
	if err != nil {
		t.Fatalf("could not parse '%s' as an int: %+v", backendUrl.Port(), err)
	}
	proxy.config.Services[0].Hosts = []config.HostPort{{
		Address: backendUrl.Hostname(),
		Port:    int(parsedPort),
	}}

	proxy.reverseProxy = make(map[string]*httputil.ReverseProxy)
	proxy.reverseProxy["my-service.my-company.com"] = NewRandomBackendReverseProxy(
		proxy.config.Services[0].Hosts,
	)

	// Start the proxy, connect, verify we get the correct response
	ts := httptest.NewServer(http.HandlerFunc(proxy.handler))
	defer ts.Close()

	resp, err := http.Get(fmt.Sprintf("%s/?s=my-service.my-company.com", ts.URL))
	if err != nil {
		t.Fatal(err)
	}
	result, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("could not read response body: %+v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if strings.TrimSpace(string(result)) != backendResp {
		t.Fatalf("got '%s', want '%s' as response body", result, "service not found")
	}
}
