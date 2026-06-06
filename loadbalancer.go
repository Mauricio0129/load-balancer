package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync/atomic"
	"time"
)

func startLoadBalancer(config_data *Config) {
	fmt.Printf("Starting load balancer on %s:%s\n", config_data.Host, config_data.Port)
	handler := CreateHandler(config_data) // initialize the handler with the configuration

	go handler.StartHealthChecks()

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

func verifyrequest(handler *Handler, domain string, w http.ResponseWriter) bool {
	_, ok := handler.config.Backends[domain]
	if !ok {
		http.Error(w, "503 Service Unavailable: No backends available for 1 the requested host", http.StatusServiceUnavailable)
	}
	return ok
}

func roundRobinLoadBalancer(handler *Handler, r *http.Request, w http.ResponseWriter) {
	hostKey := r.Host // extract the host from the incoming request to determine which backend pool to use

	if !verifyrequest(handler, hostKey, w) {
		return // Stop execution here because the helper already sent the 503 error!
	}

	perFlightSnapshot := handler.backends.Load()
	perFlightBackends := perFlightSnapshot.(map[string][]string)

	counterPointer := handler.poolCounters[hostKey]
	localCounter := atomic.AddInt64(counterPointer, 1) - 1 // atomically increment the counter and get the current value

	lenghtOfBackends := len(perFlightBackends[hostKey]) // get the number of backends for the requested host to calculate the index of the backend to use

	backend := perFlightBackends[hostKey][localCounter%int64(lenghtOfBackends)] // select the backend based on the counter value and the number of backends
	fmt.Printf(r.URL.Host)
	url := "http://" + backend + r.URL.Path // construct the URL for the selected backend
	handler.dispatchRequest(r, w, url)      // dispatch the request to the selected backend
}

func leastConnectionsLoadBalancer(handler *Handler, r *http.Request, w http.ResponseWriter) {
	hostKey := r.Host

	if !verifyrequest(handler, hostKey, w) {
		return // Stop execution here because the helper already sent the 503 error!
	}

	data := handler.connections.Load()

	snapshot, ok := data.(map[string]map[string]*int64)
	if !ok {
		http.Error(w, "503 Service Unavailable: Internal server error", http.StatusInternalServerError)
		return
	}

	backends, ok := snapshot[hostKey]
	if !ok || len(backends) == 0 {
		http.Error(w, "503 Service Unavailable: No backends available for the requested host", http.StatusServiceUnavailable)
		return
	}

	var server string
	var couunter *int64
	for backend, connections := range backends {
		if server == "" {
			server = backend
			couunter = connections
		} else if *connections < *couunter {
			server = backend
			couunter = connections
		}
	}

	atomic.AddInt64(couunter, 1)
	defer func() { // ensure that the connection count is decremented after the request is handled, even if an error occurs
		atomic.AddInt64(couunter, -1)
	}()

	url := "http://" + server + r.URL.Path
	handler.dispatchRequest(r, w, url)
}

// --- Health check logic  ---
func (h *Handler) StartHealthChecks() {
	// to do: implement health check logic to monitor backend servers and update the list of available backends accordingly
}

// --- Inittialization logic  ---
func CreateHandler(config_data *Config) *Handler {

	handler := &Handler{
		config:            config_data,
		poolCounters:      make(map[string]*int64),                    // as integers do form part of dict we need a more stable reference
		concurrency_limit: make(chan struct{}, config_data.Max_queue), // Example concurrency limit of 100
	}

	var numberOfBackends int
	var perBackendConnectionsCounters map[string]map[string]*int64

	if config_data.Mode == 1 {
		perBackendConnectionsCounters = make(map[string]map[string]*int64)
	}

	for key, value := range config_data.Backends {
		handler.poolCounters[key] = new(int64)
		numberOfBackends += len(value)

		if config_data.Mode == 1 {
			perBackendConnectionsCounters[key] = make(map[string]*int64)
			for _, backend := range value {
				perBackendConnectionsCounters[key][backend] = new(int64)
			}
		}

	}

	handler.connections.Store(perBackendConnectionsCounters)
	handler.backends.Store(config_data.Backends)

	// to do: Move out the HTTP client initialization to a separate function and implement connection pooling and timeouts based on the configuration settings
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
