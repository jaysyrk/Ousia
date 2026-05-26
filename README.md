# Ousia

A High-Performance, Dynamic L7 Edge Gateway and Service Mesh in Go.

## What is Ousia?

Ousia empowers small programs on lightweight servers to reliably handle the same massive traffic loads as giant enterprise systems by leveraging advanced load balancing, intelligent routing, and built-in resiliency. It handles inbound edge traffic and transparently manages service-to-service communication via lightweight sidecar proxies, providing load balancing, resiliency, and observability out of the box.

## Key Features

* **Dynamic L7 Routing**: Map traffic to upstream pools based on exact paths, prefixes, HTTP methods, and Host headers with priority-based resolution.
* **Pluggable Load Balancing**: Out-of-the-box support for Round Robin, Weighted Round Robin, Least Connections, and IP Hash algorithms.
* **Lock-Free Atomic State Engine**: Sustains 2,100+ RPS with zero Mutex bottlenecks or Garbage Collector pauses via hardware-level atomics, 256-bucket memory sharding, and `sync.Pool` object recycling.
* **Resiliency Middleware**: Built-in Circuit Breakers, Retries, Rate Limiting, and Timeouts to keep your services online during downstream failures.
* **Service Mesh Sidecars**: Transparent proxies that automatically handle service discovery, routing, and telemetry injection.
* **Native Observability**: Automatic Prometheus metrics aggregation, distributed trace ID propagation, and asynchronous batch-flushed logging.
* **Zero-Friction CLI**: Manage the entire cluster dynamically using the `ousiactl` command-line tool.

## Configuration

Ousia uses a centralized configuration file to define upstreams, routing policies, and mesh topologies. By default, the control plane and gateway look for an `ousia.yaml` file in the working directory.

```yaml
# Example snippet from ousia.yaml
gateway:
  http_port: 8080
control_plane:
  grpc_port: 50051
```

## How to Use It

### 1. Quickstart (Local Cluster)

Ousia comes with a ready-to-run Docker Compose environment that spins up the Control Plane / Edge Gateway, a Prometheus metrics server, and two demo microservices with attached sidecars.

```bash
# Clone the repository
git clone https://github.com/jaysyrk/Ousia.git
cd Ousia

# Start the Service Mesh via Docker
docker-compose up -d --build
```

#### Native Local Testing (Alternative)
If you prefer running the binaries directly outside of containers, ensure your config pathing is set and execute:
```bash
# Start Control Plane
./control-plane

# Start Edge Gateway (In a new terminal)
./gateway
```

### 2. The ousiactl CLI

Use the included CLI to dynamically interact with the Control Plane and manage your mesh.

```bash
# Build the CLI via the Go workspace
go build -o ousiactl ./cmd/ousiactl

# Check gateway health status
./ousiactl status

# View registered sidecars
./ousiactl mesh list

# View aggregated mesh metrics
./ousiactl metrics

# Add a new traffic route
./ousiactl route add --id web-route --virtual-host example.com --upstream web-pool
```

### 3. Development & Testing

Ousia has a robust, zero-dependency unit test suite covering the core L7 router, load balancers, and middleware chain.

```bash
go test ./... -v
```

## License

This project is licensed under the MIT License.
