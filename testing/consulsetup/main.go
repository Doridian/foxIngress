package main

import (
	"flag"
	"github.com/hashicorp/consul/api"
	"time"
)

func main() {
	time.Sleep(10 * time.Second)
	var url string
	flag.StringVar(&url, "url", "127.0.0.1:8500", "Url pointing to consul cluster")
	flag.Parse()
	config := api.DefaultConfig()
	config.Address = url
	client, err := api.NewClient(config)
	if err != nil {
		panic(err)
	}

	// Get a handle to the KV API
	kv := client.KV()

	// PUT a new KV pair
	p := &api.KVPair{Key: "http/test.localhost:3999", Value: []byte("httpapp:8080")}
	_, err = kv.Put(p, nil)
	if err != nil {
		panic(err)
	}
	p = &api.KVPair{Key: "https/test.localhost", Value: []byte("httpsapp:8081")}
	_, err = kv.Put(p, nil)
	if err != nil {
		panic(err)
	}
}
