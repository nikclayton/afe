// Package config provides types and functions to process the YAML
// configuration file.
package config

import (
	"fmt"
	"io/ioutil"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// A host:port pair for a service.
type HostPort struct {
	Address string
	Port    int
}

// String returns a "host:port" string for the HostPort.
func (hp HostPort) String() string {
	return fmt.Sprintf("%s:%d", hp.Address, hp.Port)
}

// A service consists of a name, a domain, and an array of
// host:port pairs that provide that service.
type Service struct {
	Name   string
	Domain string
	Hosts  []HostPort
}

// A proxy consists of the host:port that the proxy should
// listen on, and details of the services it proxies for.
type Proxy struct {
	Listen   HostPort
	Services []Service
}

// The complete proxy configuration.
type ProxyConfig struct {
	Proxy
}

// Copy performs a deep copy of the ProxyConfig.
func (pc ProxyConfig) Copy(to *ProxyConfig) {
	*to = ProxyConfig{}
	to.Listen = pc.Listen
	for _, service := range pc.Services {
		s := Service{
			Name:   service.Name,
			Domain: service.Domain,
		}
		for _, host := range service.Hosts {
			h := HostPort{
				Address: host.Address,
				Port:    host.Port,
			}
			s.Hosts = append(s.Hosts, h)
		}
		to.Services = append(to.Services, s)
	}
}

// ParseConfigFromFile parses the YAML configuration from filename in to
// the provided ProxyConfig.
func ParseConfigFromFile(filename string, config *ProxyConfig) error {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return errors.Wrapf(err, "read from %s failed", filename)
	}

	return ParseConfig(content, config)
}

// ParseConfig parses the YAML configuration from data in to the
// provided ProxyConfig.
func ParseConfig(data []byte, config *ProxyConfig) error {
	if err := yaml.Unmarshal(data, config); err != nil {
		return errors.Wrap(err, "unmarshalling failed")
	}

	return nil
}

// ValidateConfig verifies the configuration appears sensible. If not it
// returns one or more errors identified in the configuration.
func ValidateConfig(config *ProxyConfig) []error {
	var errs []error
	// if config.Listen.Address == "" {
	//
	// }
	//
	// This is sometimes OK, e.g., to bind to whatever the host's first IP
	// is without caring what it is.

	if config.Listen.Port == 0 {
		errs = append(errs, errors.New("Listen Port is not set"))
	}

	if len(config.Services) == 0 {
		errs = append(errs, errors.New("No services have been defined"))
	}

	for i, service := range config.Services {
		if service.Name == "" {
			errs = append(errs, errors.Errorf("The service at index %d has no name", i))
			continue // No sense checking other parts, can't report the name
		}

		if service.Domain == "" {
			errs = append(errs, errors.Errorf("Service %s has no domain", service.Name))
		}

		if len(service.Hosts) == 0 {
			errs = append(errs, errors.Errorf("Service %s has no hosts", service.Name))
		}

		for j, host := range service.Hosts {
			if host.Address == "" {
				errs = append(errs, errors.Errorf("The %d host in service %s has no address", j, service.Name))
			}

			if host.Port == 0 {
				errs = append(errs, errors.Errorf("The %d host in service %s has no port", j, service.Name))
			}
		}
	}

	return errs
}
