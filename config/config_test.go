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

func TestValidateConfig(t *testing.T) {
	goldenConfig := ProxyConfig{
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

	checkErr := func(errs []error, expN int, msg string) {
		if len(errs) != expN {
			t.Errorf("got %d errors, want %d %+v", len(errs), expN, errs)
			return
		}
		if expN == 0 {
			return
		}
		for _, err := range errs {
			if err.Error() == msg {
				return
			}
		}
		t.Errorf("did not find '%s' in errs", msg)
		t.Error(errs)
	}

	testConfig := ProxyConfig{}

	// Check for single errors in the configuration
	goldenConfig.Copy(&testConfig)
	testConfig.Listen.Address = ""
	errs := ValidateConfig(&testConfig)
	checkErr(errs, 0, "")

	goldenConfig.Copy(&testConfig)
	testConfig.Listen.Port = 0
	errs = ValidateConfig(&testConfig)
	checkErr(errs, 1, "Listen Port is not set")

	goldenConfig.Copy(&testConfig)
	testConfig.Services = []Service{}
	errs = ValidateConfig(&testConfig)
	checkErr(errs, 1, "No services have been defined")

	goldenConfig.Copy(&testConfig)
	testConfig.Services[0].Name = ""
	errs = ValidateConfig(&testConfig)
	checkErr(errs, 1, "The service at index 0 has no name")

	goldenConfig.Copy(&testConfig)
	testConfig.Services[0].Domain = ""
	errs = ValidateConfig(&testConfig)
	checkErr(errs, 1, "Service my-service has no domain")

	goldenConfig.Copy(&testConfig)
	testConfig.Services[0].Hosts = []HostPort{}
	errs = ValidateConfig(&testConfig)
	checkErr(errs, 1, "Service my-service has no hosts")

	goldenConfig.Copy(&testConfig)
	testConfig.Services[0].Hosts[0].Address = ""
	errs = ValidateConfig(&testConfig)
	checkErr(errs, 1, "The 0 host in service my-service has no address")

	goldenConfig.Copy(&testConfig)
	testConfig.Services[0].Hosts[0].Port = 0
	errs = ValidateConfig(&testConfig)
	checkErr(errs, 1, "The 0 host in service my-service has no port")

	// Check multiple errors are reported
	goldenConfig.Copy(&testConfig)
	testConfig.Listen.Port = 0
	testConfig.Services[0].Hosts[0].Port = 0
	errs = ValidateConfig(&testConfig)
	checkErr(errs, 2, "Listen Port is not set")
	checkErr(errs, 2, "The 0 host in service my-service has no port")
}
