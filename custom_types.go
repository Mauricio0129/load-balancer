package main

import (
	"net/http"
)

type Config struct {
	Host            string              `json:"host"`
	Port            string              `json:"port"`
	Backends        map[string][]string `json:"backends"`
	Tls_config      Tls                 `json:"tls"`
	Timeouts_config Timeouts            `json:"timeouts"`
	Mode            int                 `json:"mode"`
	Max_queue       int                 `json:"max_queue"`
}

type Timeouts struct {
	ReadHeader   int `json:"readheader_timeout"`
	WriteTimeout int `json:"write_timeout"`
}

type Tls struct {
	Enabled  bool   `json:"enabled"`
	CertFile string `json:"certfile"`
	KeyFile  string `json:"keyfile"`
}

type carryConfig struct {
	config *Config
}

type Handler struct {
	config            *Config
	poolCounters      map[string]*int64
	concurrency_limit chan struct{}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	select {
	case h.concurrency_limit <- struct{}{}:
		defer func() { <-h.concurrency_limit }()

		h.setAdaptiveDeadlines(w, r)

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

	default:

		h.rejectTraffic(w)
		return
	}
}
