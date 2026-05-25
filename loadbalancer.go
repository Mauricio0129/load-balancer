package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func startLoadBalancer(config *Config) {
	fmt.Printf("Starting load balancer on %s:%s\n", config.Host, config.Port)

	http.Handle("/", &Handler{config: config})

	if config.Tls_config.Enabled == true {
		// Start HTTPS load balancer
		// to do: implement TLS support
	} else {
		server := &http.Server{
			Addr:              config.Host + ":" + config.Port,
			ReadHeaderTimeout: time.Second * time.Duration(config.Timeouts_config.ReadHeader),
			WriteTimeout:      time.Second * time.Duration(config.Timeouts_config.WriteTimeout),
		}
		log.Fatal(server.ListenAndServe())

	}
}
