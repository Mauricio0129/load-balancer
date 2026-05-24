package main

type Config struct {
	Host       string              `json:"host"`
	Port       string              `json:"port"`
	Backends   map[string][]string `json:"backends"`
	Tls_config Tls                 `json:"tls"`
}

type Tls struct {
	Enabled  bool   `json:"enabled"`
	CertFile string `json:"certfile"`
	KeyFile  string `json:"keyfile"`
}
