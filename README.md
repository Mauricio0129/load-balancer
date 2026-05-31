# Custom High-Concurrency Layer 7 Load Balancer

A highly concurrent Layer 7 reverse proxy and load balancer written in Go. Built as a personal project to learn systems programming and Go concurrency patterns. It ingests microservice cluster layouts via JSON configuration, manages traffic spikes using an event-driven adaptive deadline queue, and routes traffic through custom-tuned connection pools using thread-safe load balancing algorithms.

---

## 🚀 Key Features

- **Dynamic Environment Routing** — Supports path/cluster-based routing split across different environments (e.g., `www` data vs `api` endpoints).
- **Non-Blocking Channel Gatekeeper** — Implements a strict capacity boundary via weightless `struct{}{}` buffered channels rather than heavy worker pools, keeping CPU usage low.
- **Timed Backoff Waiting Room** — Replaces instant traffic rejections with a thread-safe `time.After` channel race, giving burst traffic a microsecond-level window to acquire a slot.
- **Context Pass-Through Protection** — Utilizes `http.NewRequestWithContext` to link client browser sockets directly to backend pipes, tearing down dead upstream processes instantly if a user disconnects.
- **Zero-Copy Byte Mirroring** — Streams data payloads via raw `[]byte` memory blocks, transparently reflecting backend status codes and headers with minimal allocation overhead.

---

## 💻 Getting Started

### Prerequisites

Make sure Go is installed. Developed and tested on:

```bash
$ go version
go version go1.26.0 darwin/arm64
```

### Run the Load Balancer

From the root directory:

```bash
$ go run .
```

---

## ⚙️ Configuration — `config.json`

All behavior is configured through a single JSON file at the root of the project:

```json
{
    "host": "localhost",
    "port": "8080",
    "backends": {
        "www": ["localhost:9000", "localhost:9001", "localhost:9002"],
        "api": ["localhost:9050", "localhost:9051", "localhost:9052"]
    },
    "tls": {
        "enabled": false,
        "certfile": "./somefile",
        "keyfile": "./somekey"
    },
    "timeouts": {
        "readheader_timeout": 7,
        "write_timeout": 7,
        "client_timeout": 7
    },
    "max_queue": 100,
    "max_idle_conns": 100,
    "mode": 1
}
```

### Field Breakdown

| Field | Description |
|---|---|
| `backends` | Map of environment clusters holding arrays of target physical server network locations. |
| `max_queue` | Maximum concurrent request channel size before the timed backoff waiting room triggers. |
| `max_idle_conns` | Base pooling limit used to calculate total safe connection caching sizes globally. |
| `tls.enabled` | Toggle TLS termination at the proxy. Provide `certfile` and `keyfile` paths when enabled. |
| `timeouts.*` | Per-phase HTTP timeout values in seconds (read header, write, client). |
| `mode` | Load balancing algorithm selector — see options below. |

### Load Balancing Modes

| Mode | Algorithm | Description |
|---|---|---|
| `0` | Atomic Round Robin | Distributes requests sequentially across all backends using an atomic counter. |
| `1` | Atomic Least Connections | Routes each request to the backend currently handling the fewest active connections. |

---

## 🤖 AI Usage Disclosure

Gemini was used as a sounding board throughout this project for:

- Structural design decisions regarding Go's runtime scheduling model (`gopark`)
- Validating type-assertions and structural memory-leak protections
- Debugging concurrent resource pooling math
