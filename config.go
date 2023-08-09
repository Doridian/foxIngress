package main

import (
	"errors"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

var backendsHttp map[string]*BackendInfo
var backendsHttps map[string]*BackendInfo

type BackendProtocol int

const (
	PROTO_HTTP BackendProtocol = iota
	PROTO_HTTPS
)

const HOST_DEFAULT = "__default__"

type configBackend struct {
	Host          string `yaml:"host"`
	ProxyProtocol bool   `yaml:"proxy_protocol"`
	Port          int    `yaml:"port"`
}

type configHost struct {
	Http  configBackend `yaml:"http"`
	Https configBackend `yaml:"https"`
}

type configBase struct {
	Defaults struct {
		Backends configHost `yaml:"backends"`
	} `yaml:"defaults"`
	Hosts map[string]configHost `yaml:"hosts"`
}

type BackendInfo struct {
	Host          string
	ProxyProtocol bool
	Port          int
}

func GetBackend(hostname string, protocol BackendProtocol) (*BackendInfo, error) {
	var backends map[string]*BackendInfo
	switch protocol {
	case PROTO_HTTP:
		backends = backendsHttp
	case PROTO_HTTPS:
		backends = backendsHttps
	default:
		return nil, errors.New("invalid protocol")
	}
	backend, ok := backends[hostname]
	if !ok {
		return backends[HOST_DEFAULT], nil
	}
	return backend, nil
}

func backendConfigFromConfigHost(host *configBackend, port int) *BackendInfo {
	return &BackendInfo{
		Host:          host.Host,
		Port:          port,
		ProxyProtocol: host.ProxyProtocol,
	}
}

func LoadConfig() {
	file, err := os.Open(os.Getenv("CONFIG_FILE"))
	if err != nil {
		log.Panicf("Could not open config file: %v", err)
	}
	decoder := yaml.NewDecoder(file)
	var config configBase
	decoder.Decode(&config)

	backendsHttp = make(map[string]*BackendInfo)
	backendsHttps = make(map[string]*BackendInfo)

	for host, hostConfig := range config.Hosts {
		portHttp := hostConfig.Http.Port
		if portHttp == 0 {
			portHttp = config.Defaults.Backends.Http.Port
		}
		if portHttp > 0 {
			backendsHttp[host] = backendConfigFromConfigHost(&hostConfig.Http, portHttp)
		}

		portHttps := hostConfig.Https.Port
		if portHttps == 0 {
			portHttps = config.Defaults.Backends.Https.Port
		}
		if portHttps > 0 {
			backendsHttps[host] = backendConfigFromConfigHost(&hostConfig.Https, portHttps)
		}
	}
}
