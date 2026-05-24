package main

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
