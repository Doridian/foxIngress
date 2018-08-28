package main

import (
	"github.com/hashicorp/consul/api"
	"testing"
)

func TestGetBackendExist(t *testing.T) {
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		t.Error("Could not setup consul client")
	}

	kv := client.KV()
	output, err := getBackend("andrew.localhost:4000", "https/", kv)
	if output != "54.241.136.17:8675" {
		t.Error("incorrect output from backend. Expected: 54.241.136.17:8675  Recieved ", output)
	}
}

func TestGetBackendDoesNotExist(t *testing.T) {
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		t.Error("Could not setup consul client")
	}

	kv := client.KV()
	output, err := getBackend("fake.localhost:4000", "https/", kv)
	if output != "" {
		t.Error("incorrect output from backend. Expected: empty Recieved ", output)
	}
}
