package main

import "os"

var backends map[string]string

func GetBackend(hostname string, protocol string) (string, error) {
	backend := backends[hostname]
	if backend == "" {
		return backends["__default_"+protocol], nil
	}
	return backend, nil
}

func LoadConfig() {
	backends = make(map[string]string)
	backends["__default_http"] = os.Getenv("DEFAULT_BACKEND_HTTP")
	backends["__default_https"] = os.Getenv("DEFAULT_BACKEND_HTTPS")
}
