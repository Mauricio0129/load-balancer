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
	Maxidle_conns   int                 `json:"max_idle_conns"`
}

type Timeouts struct {
	ReadHeader    int `json:"readheader_timeout"`
	WriteTimeout  int `json:"write_timeout"`
	ClientTimeout int `json:"client_timeout"`
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
	client            *http.Client
}
