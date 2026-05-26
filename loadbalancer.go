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
		config:            config_data,
		poolCounters:      make(map[string]*int64),
		concurrency_limit: make(chan struct{}, config_data.Max_queue), // Example concurrency limit of 100
	}

	for key := range config_data.Backends {
		handler.poolCounters[key] = new(int64)
	}

	http.Handle("/", handler)

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

// --- Trafic handling logic  ---

func (h *Handler) setAdaptiveDeadlines(w http.ResponseWriter, r *http.Request) {

	controller := http.NewResponseController(w)
	minBytesPerSecond := 100 * 1024

	if r.ContentLength <= 0 {
		controller.SetReadDeadline(time.Now().Add(time.Second * 5))
	} else {
		secondsNeeded := r.ContentLength / int64(minBytesPerSecond)
		controller.SetReadDeadline(time.Now().Add(2*time.Second + (time.Duration(secondsNeeded) * time.Second)))
	}

}

func (h *Handler) dispatchRequest(r *http.Request, w http.ResponseWriter) {
	// to do: implement request dispatching logic based on load balancing mode
}

func (h *Handler) rejectTraffic(w http.ResponseWriter) {
	// to do: implement traffic rejection logic (e.g., return 503 Service Unavailable)
}

// --- Trafic balancing logic  ---

func roundRobinLoadBalancer(handler *Handler, r *http.Request, w http.ResponseWriter) {
	// to do: implement round robin load balancing
}

func leastConnectionsLoadBalancer(handler *Handler, r *http.Request, w http.ResponseWriter) {
	// to do: implement least connections load balancing
}
