# Ousia Architecture

This diagram maps out the Ousia architecture, including the gateway, sidecars, and secured traffic flows.

```mermaid
graph TD
    %% Define Styles
    classDef user fill:#e1f5fe,stroke:#0288d1,stroke-width:2px,color:#000
    classDef gateway fill:#e8f5e9,stroke:#388e3c,stroke-width:2px,color:#000
    classDef controlplane fill:#f3e5f5,stroke:#7b1fa2,stroke-width:2px,color:#000
    classDef sidecar fill:#fff3e0,stroke:#f57c00,stroke-width:2px,color:#000
    classDef service fill:#eeeeee,stroke:#616161,stroke-width:2px,color:#000
    classDef external fill:#ffebee,stroke:#d32f2f,stroke-width:2px,color:#000

    %% External Actors
    User((Client User)):::user
    Admin((Admin/DevOps)):::user
    Prometheus[(Prometheus)]:::external

    %% Gateway Components
    subgraph Ousia Gateway
        Router["Router\n(Virtual Hosts, Prefix Match)"]:::gateway
        RateLimiter["Rate Limiter\n(Token Bucket)"]:::gateway
        Balancer["Load Balancers\n(WRR, LeastConn)"]:::gateway
        
        subgraph Control Plane
            AdminAPI["Admin API :9000\n(Auth Protected)"]:::controlplane
            Registry["Mesh Registry\n(Instances & Heartbeats)"]:::controlplane
            Watcher["Config Watcher\n(ousia.yaml)"]:::controlplane
        end
    end

    %% Mesh Node A
    subgraph Mesh Node A
        SidecarInA["Sidecar Inbound :15000\n(WASM Plugin)"]:::sidecar
        SidecarOutA["Sidecar Outbound :15001"]:::sidecar
        ServiceA["Service A :80\n(Local App)"]:::service
    end

    %% Mesh Node B
    subgraph Mesh Node B
        SidecarInB["Sidecar Inbound :15000\n(WASM Plugin)"]:::sidecar
        SidecarOutB["Sidecar Outbound :15001"]:::sidecar
        ServiceB["Service B :80\n(Local App)"]:::service
    end

    %% Connections
    User -- "HTTP :8080\nHTTPS :8443" --> RateLimiter
    RateLimiter --> Router
    Router --> Balancer
    Balancer -- "Proxies Traffic" --> SidecarInA
    Balancer -- "Proxies Traffic" --> SidecarInB

    %% Admin flows
    Admin -- "Bearer Token\nConfig Updates" --> AdminAPI
    AdminAPI --> Router
    AdminAPI --> Balancer
    Watcher -- "SHA-256 Hash\nConfig Hot Reload" --> Router

    %% Sidecar internals
    SidecarInA -- "Local proxy" --> ServiceA
    SidecarInB -- "Local proxy" --> ServiceB
    
    ServiceA -- "Service-to-Service" --> SidecarOutA
    SidecarOutA -- "Discovers via Control Plane" --> SidecarInB

    %% Control Plane interactions
    SidecarOutA -. "Heartbeats & Registration" .-> Registry
    SidecarOutB -. "Heartbeats & Registration" .-> Registry

    %% Observability
    Prometheus -. "Scrapes /metrics" .-> AdminAPI
    Prometheus -. "Scrapes /stats\n(Auth Protected)" .-> SidecarInA
```

### What this diagram highlights:
- **The Data Plane:** Shows how external client traffic enters the Gateway, passes through the Rate Limiter, gets routed, balanced, and forwarded to the Sidecar Inbounds (where the WASM plugin runs).
- **The Control Plane:** Visualizes the `Admin API`, `Mesh Registry`, and `Config Watcher` driving dynamic updates to the Router and Balancer.
- **Service-to-Service:** Shows how `Service A` talks to its Outbound Sidecar, which discovers the route to `Service B`'s Inbound Sidecar.
- **Security:** Explicitly calls out the Auth Protection on the Admin API and Sidecar `/stats`, as well as the WASM Plugin and SHA-256 hash checks.
