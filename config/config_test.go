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

func TestHostPortString(t *testing.T) {
	var tests = []struct {
		in  HostPort
		out string
	}{
		{HostPort{Address: "127.0.0.1", Port: 8080}, "127.0.0.1:8080"},
		{HostPort{Address: "127.0.0.1"}, "127.0.0.1:0"},
		{HostPort{Port: 8081}, ":8081"},
	}

	for _, tt := range tests {
		s := tt.in.String()
		if s != tt.out {
			t.Errorf("got %q, want %q", s, tt.out)
		}
	}
}
