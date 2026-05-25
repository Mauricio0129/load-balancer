package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func startLoadBalancer(config_data *Config) {
	fmt.Printf("Starting load balancer on %s:%s\n", config_data.Host, config_data.Port)

	handler := &Handler{
		config:       config_data,
		poolCounters: make(map[string]*int64),
	}

	for key := range config_data.Backends {
		handler.poolCounters[key] = new(int64)
	}

	http.Handle("/", &Handler{config: config_data})

	if config_data.Tls_config.Enabled == true {
		// Start HTTPS load balancer
		// to do: implement TLS support
	} else {
		server := &http.Server{
			Addr:              config_data.Host + ":" + config_data.Port,
			ReadHeaderTimeout: time.Second * time.Duration(config_data.Timeouts_config.ReadHeader),
			WriteTimeout:      time.Second * time.Duration(config_data.Timeouts_config.WriteTimeout),
		}
		log.Fatal(server.ListenAndServe())

	}
}

func roundRobinLoadBalancer(handler *Handler, r *http.Request, w http.ResponseWriter) {
	// to do: implement round robin load balancing
}

func leastConnectionsLoadBalancer(handler *Handler, r *http.Request, w http.ResponseWriter) {
	// to do: implement least connections load balancing
}
