# Ousia

**A High-Performance, Dynamic L7 Edge Gateway and Service Mesh in Go.**

## What is Ousia?
Ousia is a lightweight, cloud-native Service Mesh and Edge Gateway designed for developers who want the power of advanced traffic management without the massive operational overhead of larger meshes. 

It handles inbound edge traffic and transparently manages service-to-service communication via lightweight sidecar proxies, providing load balancing, resiliency, and observability out of the box.

## Key Features
- **Dynamic L7 Routing**: Map traffic to upstream pools based on exact paths, prefixes, HTTP methods, and Host headers with priority-based resolution.
- **Pluggable Load Balancing**: Out-of-the-box support for Round Robin, Weighted Round Robin, Least Connections, and IP Hash algorithms.
- **Resiliency Middleware**: Built-in Circuit Breakers, Retries, Rate Limiting, and Timeouts to keep your services online during downstream failures.
- **Service Mesh Sidecars**: Transparent proxies that automatically handle service discovery, routing, and telemetry injection.
- **Native Observability**: Automatic Prometheus metrics aggregation and distributed trace ID propagation.
- **Zero-Friction CLI**: Manage the entire cluster dynamically using the `ousiactl` command-line tool.

## How to Use It

### 1. Quickstart (Local Cluster)
Ousia comes with a ready-to-run Docker Compose environment that spins up the Control Plane / Edge Gateway, a Prometheus metrics server, and two demo microservices with attached sidecars.

```bash
# Clone the repository
git clone https://github.com/jaysyrk/Ousia.git
cd Ousia

# Start the Service Mesh
docker-compose up -d --build
```

### 2. The `ousiactl` CLI
Use the included CLI to dynamically interact with the Control Plane and manage your mesh.

```bash
# Build the CLI
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
