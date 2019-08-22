package main

import (
	"afe/config"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
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
