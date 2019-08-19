package config

import (
	"testing"

	"github.com/go-test/deep"
)

func TestParseConfig(t *testing.T) {
	var actualConfig ProxyConfig

	var yaml = `proxy:
  listen:
    address: "127.0.0.1"
    port: 8080

  services:
    - name: my-service
      domain: my-service.my-company.com
      hosts:
        - address: "127.0.0.1"
          port: 9090
        - address: "127.0.0.1"
          port: 9091
`
	expectedConfig := ProxyConfig{
		Proxy{
			Listen: HostPort{
				Address: "127.0.0.1",
				Port:    8080,
			},
			Services: []Service{{
				Name:   "my-service",
				Domain: "my-service.my-company.com",
				Hosts: []HostPort{{
					Address: "127.0.0.1",
					Port:    9090,
				}, {
					Address: "127.0.0.1",
					Port:    9091,
				}},
			}},
		},
	}

	if err := ParseConfig([]byte(yaml), &actualConfig); err != nil {
		t.Error("valid config failed to parse", err)
	}

	if diff := deep.Equal(actualConfig, expectedConfig); diff != nil {
		t.Error(diff)
	}
}
