# Custom High-Concurrency Layer 7 Load Balancer

A highly concurrent Layer 7 reverse proxy and load balancer written in Go. Built as a personal project to learn systems programming and Go concurrency patterns. It ingests microservice cluster layouts via JSON configuration, manages traffic spikes using an event-driven adaptive deadline queue, and routes traffic through custom-tuned connection pools using thread-safe load balancing algorithms.

---

## 🚀 Key Features

- **Host-Based Environment Routing** — Routes incoming traffic to isolated backend clusters based on the request's `Host` header (e.g., `api.localhost:8080` vs `www.localhost:8080`), enabling true multi-environment routing without path rewriting.
- **Non-Blocking Channel Gatekeeper** — Implements a strict capacity boundary via weightless `struct{}{}` buffered channels rather than heavy worker pools, keeping CPU usage low.
- **Timed Backoff Waiting Room** — Replaces instant traffic rejections with a thread-safe `time.After` channel race, giving burst traffic a 25ms window to acquire a slot before returning a 503.
- **Adaptive Request Deadlines** — Sets read deadlines dynamically based on `Content-Length`, giving large payloads proportionally more time while keeping a hard 5s cap on requests with unknown body sizes — mitigating slowloris-style attacks.
- **Context Pass-Through Protection** — Utilizes `http.NewRequestWithContext` to link client sockets directly to backend pipes, tearing down upstream connections instantly if a user disconnects.
- **Lock-Free Backend Snapshots** — Stores backend lists and per-backend connection counters in `atomic.Value`, allowing goroutines to read routing state without mutexes or blocking.
- **Dynamic Connection Pool Sizing** — Computes `MaxIdleConns` at startup as `(numberOfBackends × max_idle_conns) + numberOfBackends`, ensuring each backend gets a dedicated idle connection budget without starving others.
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
        "api.localhost:8080": ["localhost:9000", "localhost:9001", "localhost:9002"],
        "www.localhost:8080": ["localhost:9050", "localhost:9051", "localhost:9052"]
    },
    "tls": {
        "enabled": false,
        "certfile": "./cert.pem",
        "keyfile": "./key.pem"
    },
    "timeouts": {
        "readheader_timeout": 7,
        "write_timeout": 7,
        "client_timeout": 7
    },
    "max_queue": 100,
    "max_idle_conns": 100,
    "mode": 0
}
```

### Field Breakdown

| Field | Description |
|---|---|
| `backends` | Map of `Host` header keys to arrays of backend server addresses. Each key is matched against the incoming request's `Host` header. |
| `max_queue` | Maximum concurrent request channel size before the timed backoff waiting room triggers. |
| `max_idle_conns` | Per-host idle connection budget. Total pool size is computed as `(backends × max_idle_conns) + backends`. |
| `tls.enabled` | Toggle TLS termination at the proxy. Provide `certfile` and `keyfile` paths when enabled. |
| `timeouts.*` | Per-phase HTTP timeout values in seconds (read header, write, client). |
| `mode` | Load balancing algorithm selector — see options below. |

### Load Balancing Modes

| Mode | Algorithm | Description |
|---|---|---|
| `0` | Atomic Round Robin | Distributes requests sequentially across all backends using a per-cluster atomic counter. |
| `1` | Atomic Least Connections | Routes each request to the backend currently handling the fewest active connections, tracked with per-backend atomic counters. |

---

## 🤖 AI Usage Disclosure

Gemini was used as a sounding board throughout this project for:

- Structural design decisions regarding Go's runtime scheduling model (`gopark`)
- Validating type-assertions and structural memory-leak protections
- Debugging concurrent resource pooling math
