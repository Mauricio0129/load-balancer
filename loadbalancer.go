package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

func startLoadBalancer(config_data *Config) {
	fmt.Printf("Starting load balancer on %s:%s\n", config_data.Host, config_data.Port)
	handler := CreateHandler(config_data) // initialize the handler with the configuration

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

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	select {
	case h.concurrency_limit <- struct{}{}:
		defer func() { <-h.concurrency_limit }() // buffered channel to limit concurrency and ensure it is released after handling the request

		h.setAdaptiveDeadlines(w, r) // set adaptive deadlines based on content length to prevent slowloris attacks

		switch h.config.Mode {
		case 0:
			// Round Robin
			roundRobinLoadBalancer(h, r, w)
		case 1:
			// Least Connections
			leastConnectionsLoadBalancer(h, r, w)
		default:
			// Default to Round Robin
			roundRobinLoadBalancer(h, r, w)
		}

	// CASE 2: The Timed Waiting Room (Replaces the 'default' block)
	case <-time.After(25 * time.Millisecond): // short timeout to reject requests when the server is too busy
		h.rejectTraffic(w)
		return
	}
}

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

func (h *Handler) dispatchRequest(r *http.Request, w http.ResponseWriter, url string) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}

	r_data := bytes.NewReader(data) // make the extracted data into an io.Reader to be used in the new request

	userContext := r.Context()

	request, err := http.NewRequestWithContext(userContext, r.Method, url, r_data)
	if err != nil {
		http.Error(w, "Failed to create request for backend", http.StatusInternalServerError)
		return
	}

	response, err := h.client.Do(request) // send the request to the backend server using the custom HTTP client with connection pooling and timeouts
	if err != nil {
		http.Error(w, "Failed to forward request to backend", http.StatusBadGateway)
		return
	}

	defer response.Body.Close()

	responseData, err := io.ReadAll(response.Body)
	if err != nil {
		http.Error(w, "Failed to read response from backend", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(response.StatusCode)
	w.Write(responseData)

}

func (h *Handler) rejectTraffic(w http.ResponseWriter) {
	http.Error(w, "503 Service Unavailable: Server is too busy", http.StatusServiceUnavailable)
}

// --- Trafic balancing logic  ---

func roundRobinLoadBalancer(handler *Handler, r *http.Request, w http.ResponseWriter) {
	// to do: implement round robin load balancing
}

func leastConnectionsLoadBalancer(handler *Handler, r *http.Request, w http.ResponseWriter) {
	// to do: implement least connections load balancing
}

// --- Inittialization logic  ---
func CreateHandler(config_data *Config) *Handler {

	fmt.Printf("Starting load balancer on %s:%s\n", config_data.Host, config_data.Port)

	handler := &Handler{
		config:            config_data,
		poolCounters:      make(map[string]*int64),                    // as integers do form part of dict we need a more stable reference
		concurrency_limit: make(chan struct{}, config_data.Max_queue), // Example concurrency limit of 100
	}

	// Initialize pool counters for each backend
	var numberOfBackends int
	for key, value := range config_data.Backends {
		handler.poolCounters[key] = new(int64)
		numberOfBackends += len(value)

	}

	defaultTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		log.Fatalf("Critical: Failed to assert http.DefaultTransport to *http.Transport")

	}

	customTransport := &http.Transport{
		Proxy:       http.ProxyFromEnvironment,
		DialContext: defaultTransport.DialContext,

		ForceAttemptHTTP2:     true,
		MaxIdleConns:          numberOfBackends*config_data.Maxidle_conns + numberOfBackends,
		MaxIdleConnsPerHost:   config_data.Maxidle_conns,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	customClient := &http.Client{
		Transport: customTransport,
		Timeout:   time.Second * time.Duration(config_data.Timeouts_config.ClientTimeout),
	}

	handler.client = customClient

	return handler

}
