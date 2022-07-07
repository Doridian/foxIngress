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

type configPorts struct {
	Http  int `yaml:"http"`
	Https int `yaml:"https"`
}

type configHost struct {
	Target        string      `yaml:"target"`
	ProxyProtocol bool        `yaml:"proxy_protocol"`
	Ports         configPorts `yaml:"ports"`
}

type configBase struct {
	Defaults struct {
		Ports configPorts `yaml:"ports"`
	} `yaml:"defaults"`
	Aliases map[string]configHost `yaml:"aliases"`
	Hosts   map[string]configHost `yaml:"hosts"`
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

func tryMapHost(host *configHost, config *configBase) *configHost {
	res, ok := config.Aliases[host.Target]
	if !ok {
		return host
	}
	return &res
}

func backendConfigFromConfigHost(host *configHost, port int) *BackendInfo {
	return &BackendInfo{
		Host:          host.Target,
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

	for host, rawHostConfig := range config.Hosts {
		hostConfig := tryMapHost(&rawHostConfig, &config)

		portHttp := hostConfig.Ports.Http
		if portHttp == 0 {
			portHttp = config.Defaults.Ports.Http
		}
		if portHttp > 0 {
			backendsHttp[host] = backendConfigFromConfigHost(hostConfig, portHttp)
		}

		portHttps := hostConfig.Ports.Https
		if portHttps == 0 {
			portHttps = config.Defaults.Ports.Https
		}
		if portHttps > 0 {
			backendsHttps[host] = backendConfigFromConfigHost(hostConfig, portHttps)
		}
	}
}
