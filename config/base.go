package config

import (
	"errors"
	"fmt"
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

var config configBase

type BackendProtocol int

const (
	PROTO_HTTP BackendProtocol = iota
	PROTO_HTTPS
	PROTO_QUIC
)

func (p *BackendProtocol) String() string {
	switch *p {
	case PROTO_HTTP:
		return "HTTP"
	case PROTO_HTTPS:
		return "HTTPS"
	case PROTO_QUIC:
		return "QUIC"
	default:
		return "UNKNOWN"
	}
}

const HOST_DEFAULT = "__default__"

type BackendInfo struct {
	Host string
	Port int

	ProxyProtocol   bool
	HostPassthrough bool
	Match           string
}

func (b *BackendInfo) String() string {
	if b == nil {
		return "nil"
	}
	return fmt.Sprintf("%s:%d", b.Host, b.Port)
}

type backendInfoEncoded struct {
	Host            *string `yaml:"host"`
	Port            *int    `yaml:"port"`
	Disabled        *bool   `yaml:"disabled"`
	ProxyProtocol   *bool   `yaml:"proxy_protocol"`
	HostPassthrough *bool   `yaml:"host_passthrough"`
}

type configHost struct {
	Default  *backendInfoEncoded `yaml:"default"`
	Http     *backendInfoEncoded `yaml:"http"`
	Https    *backendInfoEncoded `yaml:"https"`
	Quic     *backendInfoEncoded `yaml:"quic"`
	Template string              `yaml:"template"`
}

type configBase struct {
	Defaults struct {
		Backends configHost `yaml:"backends"`
	} `yaml:"defaults"`
	Templates map[string]configHost `yaml:"templates"`
	Hosts     map[string]configHost `yaml:"hosts"`
	Listeners struct {
		Http       string `yaml:"http"`
		Https      string `yaml:"https"`
		Quic       string `yaml:"quic"`
		Prometheus string `yaml:"prometheus"`
	}
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

func loadBackendConfig(match string, cfgs ...*backendInfoEncoded) *BackendInfo {
	var host *string = nil
	var port *int = nil
	var disabled *bool = nil

	var proxyProto *bool = nil
	var hostPass *bool = nil

	isConfigured := false

	for _, cfg := range cfgs {
		if cfg == nil {
			continue
		}
		isConfigured = true

		if host == nil {
			host = cfg.Host
		}
		if port == nil {
			port = cfg.Port
		}
		if disabled == nil {
			disabled = cfg.Disabled
		}

		if proxyProto == nil {
			proxyProto = cfg.ProxyProtocol
		}
		if hostPass == nil {
			hostPass = cfg.HostPassthrough
		}
	}

	if !isConfigured {
		return nil
	}

	if disabled != nil && *disabled {
		return nil
	}

	if host == nil || *host == "" {
		log.Fatalf("No or empty host specified for backend %s", match)
		return nil
	}

	if port == nil || *port <= 0 || *port > 65535 {
		log.Fatalf("No or invalid port specified for backend %s", match)
		return nil
	}

	info := &BackendInfo{
		Host:  *host,
		Port:  *port,
		Match: match,
	}
	if proxyProto != nil {
		info.ProxyProtocol = *proxyProto
	}
	if hostPass != nil {
		info.HostPassthrough = *hostPass
	}
	return info
}

func Load() {
	if os.Getenv("VERBOSE") != "" {
		Verbose = true
	}

	cName := os.Getenv("CONFIG_FILE")
	if cName == "" {
		cName = "config.yml"
	}
	file, err := os.Open(cName)
	if err != nil {
		log.Fatalf("Could not open config file: %v", err)
	}
	decoder := yaml.NewDecoder(file)
	decoder.Decode(&config)

	backendsHttp = make(map[string]*BackendInfo)
	backendsHttps = make(map[string]*BackendInfo)
	backendsQuic = make(map[string]*BackendInfo)

	for match, rawHostConfig := range config.Hosts {
		hostConfig := rawHostConfig
		if rawHostConfig.Template != "" {
			hostConfig = config.Templates[hostConfig.Template]
		}

		if !wildcardsEnabled && strings.HasPrefix(match, "_.") {
			wildcardsEnabled = true
		}

		cfg := loadBackendConfig(match, hostConfig.Http, hostConfig.Default, config.Defaults.Backends.Http, config.Defaults.Backends.Default)
		if cfg != nil {
			backendsHttp[match] = cfg
		}

		cfg = loadBackendConfig(match, hostConfig.Https, hostConfig.Default, config.Defaults.Backends.Https, config.Defaults.Backends.Default)
		if cfg != nil {
			backendsHttps[match] = cfg
		}

		cfg = loadBackendConfig(match, hostConfig.Quic, hostConfig.Default, config.Defaults.Backends.Quic, config.Defaults.Backends.Default)
		if cfg != nil {
			backendsQuic[match] = cfg
		}
	}

	log.Printf("Loaded config with %d HTTP host(s), %d HTTPS host(s), %d QUIC host(s), wildard matching %v, verbose %v", len(backendsHttp), len(backendsHttps), len(backendsQuic), wildcardsEnabled, Verbose)
}

func GetHTTPAddr() string {
	return config.Listeners.Http
}

func GetHTTPSAddr() string {
	return config.Listeners.Https
}

func GetQUICAddr() string {
	return config.Listeners.Quic
}

func GetPrometheusAddr() string {
	return config.Listeners.Prometheus
}
