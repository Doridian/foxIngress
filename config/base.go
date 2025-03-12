package config

import (
	"errors"
	"log"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

var backendsHttp map[string]*BackendInfo
var backendsHttps map[string]*BackendInfo
var backendsQuic map[string]*BackendInfo
var wildcardsEnabled = false
var Verbose = false

type BackendProtocol int

const (
	PROTO_HTTP BackendProtocol = iota
	PROTO_HTTPS
	PROTO_QUIC
)

const HOST_DEFAULT = "__default__"

type BackendInfo struct {
	Host            string `yaml:"host"`
	ProxyProtocol   bool   `yaml:"proxy_protocol"`
	Port            int    `yaml:"port"`
	HostPassthrough bool   `yaml:"host_passthrough"`
}

type configHost struct {
	Default  *BackendInfo `yaml:"default"`
	Http     *BackendInfo `yaml:"http"`
	Https    *BackendInfo `yaml:"https"`
	Quic     *BackendInfo `yaml:"quic"`
	Template string       `yaml:"template"`
}

type configBase struct {
	Defaults struct {
		Backends configHost `yaml:"backends"`
	} `yaml:"defaults"`
	Templates map[string]configHost `yaml:"templates"`
	Hosts     map[string]configHost `yaml:"hosts"`
}

func findBackend(hostname string, backends map[string]*BackendInfo) (*BackendInfo, error) {
	backend, ok := backends[hostname]
	if ok {
		return backend, nil
	}

	if !wildcardsEnabled {
		return backends[HOST_DEFAULT], nil
	}

	hostSplit := strings.Split(hostname, ".")
	if hostSplit[0] == "_" {
		hostSplit = hostSplit[2:]
	} else {
		hostSplit = hostSplit[1:]
	}
	if len(hostSplit) == 0 {
		return backends[HOST_DEFAULT], nil
	}
	return findBackend("_."+strings.Join(hostSplit, "."), backends)
}

func GetBackend(hostname string, protocol BackendProtocol) (*BackendInfo, error) {
	var backends map[string]*BackendInfo
	switch protocol {
	case PROTO_HTTP:
		backends = backendsHttp
	case PROTO_HTTPS:
		backends = backendsHttps
	case PROTO_QUIC:
		backends = backendsQuic
	default:
		return nil, errors.New("invalid protocol")
	}
	return findBackend(hostname, backends)
}

func loadBackendConfig(cfg *BackendInfo, defCfg ...*BackendInfo) *BackendInfo {
	if cfg == nil {
		return nil
	}

	host := cfg.Host
	port := cfg.Port

	for _, defCfg := range defCfg {
		if defCfg == nil {
			continue
		}

		if host == "" {
			host = defCfg.Host
		}
		if port == 0 {
			port = defCfg.Port
		}
	}

	if host == "" || port <= 0 {
		return nil
	}

	return &BackendInfo{
		Host:            host,
		Port:            port,
		ProxyProtocol:   cfg.ProxyProtocol,
		HostPassthrough: cfg.HostPassthrough,
	}
}

func Load() {
	if os.Getenv("VERBOSE") != "" {
		Verbose = true
	}

	file, err := os.Open(os.Getenv("CONFIG_FILE"))
	if err != nil {
		log.Panicf("Could not open config file: %v", err)
	}
	decoder := yaml.NewDecoder(file)
	var config configBase
	decoder.Decode(&config)

	backendsHttp = make(map[string]*BackendInfo)
	backendsHttps = make(map[string]*BackendInfo)
	backendsQuic = make(map[string]*BackendInfo)

	for host, rawHostConfig := range config.Hosts {
		hostConfig := rawHostConfig
		if rawHostConfig.Template != "" {
			hostConfig = config.Templates[hostConfig.Template]
		}

		if !wildcardsEnabled && strings.HasPrefix(host, "_.") {
			wildcardsEnabled = true
		}

		cfg := loadBackendConfig(hostConfig.Http, hostConfig.Default, config.Defaults.Backends.Http)
		if cfg != nil {
			backendsHttp[host] = cfg
		}

		cfg = loadBackendConfig(hostConfig.Https, hostConfig.Default, config.Defaults.Backends.Https)
		if cfg != nil {
			backendsHttps[host] = cfg
		}

		cfg = loadBackendConfig(hostConfig.Quic, hostConfig.Default, config.Defaults.Backends.Quic)
		if cfg != nil {
			backendsQuic[host] = cfg
		}
	}

	log.Printf("Loaded config with %d HTTP host(s), %d HTTPS host(s), %d QUIC host(s), wildard matching %v, verbose %v", len(backendsHttp), len(backendsHttps), len(backendsQuic), wildcardsEnabled, Verbose)
}
