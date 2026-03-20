# Vektor

Vektor is a high-throughput, latency-optimized predictive prefetch engine for Redis. Built as a transparent RESP proxy, it utilizes a C++17 Markov Chain inference model to predict subsequent key access patterns and proactively materializes data into the cache ahead of client requests.

## Architecture

The system is strictly decoupled across an FFI boundary to maximize throughput and isolate GC overhead:

- **Proxy Layer (Go):** A highly concurrent TCP router implementing custom RESP parsing. It multiplexes client database requests downstream to standard Redis while asynchronously cloning access telemetry.
- **Inference Engine (C++17):** A deterministic, lock-free prediction engine bounding an LFU-evicted Markov Chain. State transitions and predictability probabilities are processed sequentially by a dedicated background thread.
- **Cross-Language IPC:** Telemetry is forwarded from Go to C++ via a Single-Producer Single-Consumer (SPSC) Ring Buffer utilizing raw `std::atomic` memory ordering directives and 64-byte `alignas(64)` padding to prevent false sharing limits. Validated strictly data-race free via TSAN.
- **Coordinator Pool:** Predictions exceeding the operational threshold fire native CGO callbacks back into deeply buffered Go channels. Custom thread-safe atomic pointer registries bypass Go's conservative `cgocheck` constraints. The load is structurally shed utilizing bounded Goroutine worker scaling.
- **Control Plane:** An internal gRPC server exposing operational analytics and dynamic hot-tuning configuration.

## Requirements

- Go 1.22+
- Clang (C++17)
- Docker / Docker Compose

## Build and Operation

Vektor relies on standard GNU Make targets:

- `make build`: Compiles proxy nodes and dynamically linked C++ shared libraries.
- `make test`: Executes TSAN-verified C++ tests and localized Go network suites.
- `make proto`: Regenerates gRPC and Protobuf structure bindings.
- `make run`: Orchestrates the `docker-compose` vnet bridging Vektor to an ephemeral underlying Redis node.
- `make bench`: Executes custom trace replay simulations extracting distinct p50/p90/p99 baseline measurements.

### Docker Orchestration

The active service is composed entirely via Multi-Stage compilation, flattening the binary deployments:
- `redis`: Undisturbed baseline cache store running locally on port 6379 natively bridged inside the `docker-compose` network space.
- `vektor-node`: Standalone Golang orchestration bound identically to `libvektor_engine.so`. Exposes standard proxy traffic on mapped port `6380` and exposes internal telemetry via gRPC bounded on port `9090`.

## Telemetry and Control

Telemetry configurations maintain isolated mutation paths spanning directly into the running worker queues:
- **Service Domain:** `vektor.ControlPlane`
- **Binding Address:** `:9090`
- **RPC Methods:**
  - `GetStats`: Retrieves raw prefetch hit-rate allocations, cache skips, and bounding drop levels.
  - `SetPolicy`: Submits overrides tuning decision threshold confidence scalars (`0.0`-`1.0`) natively without execution disruption.
  - `GetModelState`: Verifies aggregated tree size bounds of active C++ Markov tracked states.
