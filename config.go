package main

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

var backendsHttp map[string]string
var backendsHttps map[string]string

type BackendProtocol int

const (
	PROTO_HTTP BackendProtocol = iota
	PROTO_HTTPS
)

const HOST_DEFAULT = "__default__"

type configPorts struct {
	Http  int `yaml:"http"`
	Https int `yaml:"https"`
}

type configHost struct {
	Target string      `yaml:"target"`
	Ports  configPorts `yaml:"ports"`
}

type configBase struct {
	Defaults struct {
		Routes struct {
			Http  string `yaml:"http"`
			Https string `yaml:"https"`
		} `yaml:"routes"`
		Ports configPorts `yaml:"ports"`
	} `yaml:"defaults"`
	Hosts map[string]configHost `yaml:"hosts"`
}

func GetBackend(hostname string, protocol BackendProtocol) (string, error) {
	var backends map[string]string
	switch protocol {
	case PROTO_HTTP:
		backends = backendsHttp
	case PROTO_HTTPS:
		backends = backendsHttps
	default:
		return "", errors.New("invalid protocol")
	}
	backend := backends[hostname]
	if backend == "" {
		return backends[HOST_DEFAULT], nil
	}
	return backend, nil
}

func LoadConfig() {
	file, err := os.Open(os.Getenv("CONFIG_FILE"))
	if err != nil {
		panic(err)
	}
	decoder := yaml.NewDecoder(file)
	var config configBase
	decoder.Decode(&config)

	backendsHttp = make(map[string]string)
	backendsHttps = make(map[string]string)
	backendsHttp[HOST_DEFAULT] = config.Defaults.Routes.Http
	backendsHttps[HOST_DEFAULT] = config.Defaults.Routes.Https

	for host, hostConfig := range config.Hosts {
		portHttp := hostConfig.Ports.Http
		if portHttp <= 0 {
			portHttp = config.Defaults.Ports.Http
		}
		backendsHttp[host] = fmt.Sprintf("%s:%d", hostConfig.Target, portHttp)

		portHttps := hostConfig.Ports.Https
		if portHttps <= 0 {
			portHttps = config.Defaults.Ports.Https
		}
		backendsHttps[host] = fmt.Sprintf("%s:%d", hostConfig.Target, portHttps)
	}
}
