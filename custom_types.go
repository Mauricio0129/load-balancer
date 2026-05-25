package main

import (
	"io"
	"net/http"
	"time"
)

type Config struct {
	Host            string              `json:"host"`
	Port            string              `json:"port"`
	Backends        map[string][]string `json:"backends"`
	Tls_config      Tls                 `json:"tls"`
	Timeouts_config Timeouts            `json:"timeouts"`
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

type Handler struct {
	config       *Config
	poolCounters map[string]*int
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	controller := http.NewResponseController(w)
	minBytesPerSecond := 100 * 1024

	if r.ContentLength <= 0 {
		controller.SetReadDeadline(time.Now().Add(time.Second * 5))
	} else {
		secondsNeeded := r.ContentLength / int64(minBytesPerSecond)
		controller.SetReadDeadline(time.Now().Add(2*time.Second + (time.Duration(secondsNeeded) * time.Second)))
	}

	io.WriteString(w, r.URL.Path)

}

type carryConfig struct {
	config *Config
}
