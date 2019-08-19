// Package config provides types and functions to process the YAML
// configuration file.
package config

import (
	"io/ioutil"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// A host:port pair for a service.
type HostPort struct {
	Address string
	Port    int
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
