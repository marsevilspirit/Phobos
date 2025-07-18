# Phobos & Deimos: The Twin Moons of Mars

<div align="center">
  <pre>
  ██████╗ ██╗  ██╗ ██████╗ ██████╗  ██████╗ ███████╗
  ██╔══██╗██║  ██║██╔═══██╗██╔══██╗██╔═══██╗██╔════╝
  ██████╔╝███████║██║   ██║██████╔╝██║   ██║███████╗
  ██╔═══╝ ██╔══██║██║   ██║██╔═══╝ ██║   ██║╚════██║
  ██║     ██║  ██║╚██████╔╝██║     ╚██████╔╝███████║
  ╚═╝     ╚═╝  ╚═╝ ╚═════╝ ╚═╝      ╚═════╝ ╚══════╝
  </pre>
</div>

<p align="center">
  <strong>Phobos, the swift messenger, is a feature-rich RPC framework. Paired with its twin, <a href="https://github.com/marsevilspirit/deimos">Deimos</a>, it forms a complete microservices ecosystem for building resilient and scalable applications.</strong>
</p>

---

## Phobos & Deimos: A Symbiotic Relationship

In the cosmos of microservices, **Phobos** and **Deimos** are two celestial bodies orbiting the same planet: your application. They are designed to work in perfect harmony, each fulfilling a critical role.

*   **Phobos (Fear): The Engine of Communication.** Phobos is the RPC framework that governs the interactions between your services. It provides the speed, resilience, and intelligence needed for high-performance communication. It handles the "how": how services talk to each other, how they handle failures, and how they balance load.

*   **Deimos (Dread): The Foundation of Knowledge.** Deimos is the distributed, consistent key-value store that acts as the central nervous system for your services. It provides service discovery, configuration management, and distributed coordination. It handles the "where": where to find other services and the "what": the configuration that governs their behavior.

Together, they provide a powerful, cohesive, and elegant solution for building and managing complex microservice architectures.

## Features

### Core RPC Framework (Phobos)

*   **High-Performance RPC:** Lightweight and efficient core for low-latency, high-throughput communication.
*   **Multiple Serialization Protocols:** Supports JSON, Msgpack, and Protocol Buffers.
*   **Gzip Compression:** Reduces network bandwidth with automatic payload compression.
*   **HTTP Gateway:** Enables web clients to interact with Phobos services.
*   **Timeout Management:** Fine-grained control over request timeouts.
*   **Flexible Metadata:** Pass contextual information between services for tracing and authentication.

### Service Governance (with Deimos)

*   **Dynamic Service Discovery:** Phobos seamlessly integrates with **Deimos** to dynamically discover and communicate with services without hardcoded addresses.
*   **Intelligent Load Balancing:** Supports multiple load balancing strategies (Random, Round Robin, Consistent Hash, etc.) using service information from Deimos.
*   **Resilience and Fault Tolerance:**
    *   **Circuit Breaker:** Prevents cascading failures.
    *   **Heartbeat:** Monitors service health.
*   **Metrics and Monitoring:** Integrates with Prometheus and Grafana for deep insights into service performance.

## Quick Start: Phobos with Deimos

This example demonstrates how to run a service with Phobos that registers itself with a Deimos cluster.

### 1. Run Deimos

First, start your [Deimos](https://github.com/marsevilspirit/deimos) cluster.

### 2. Define Your Service

Define your service interface and implementation.
```go
package example

import "context"

type Args struct {
	First string
}

type Reply struct {
	Second string
}

type HelloWorld int

func (t *HelloWorld) Greet(ctx context.Context, args *Args, reply *Reply) error {
	reply.Second = "Hello " + args.First
	return nil
}
```

### 3. Create the Phobos Server

Create a server that registers the `HelloWorld` service and uses the `Deimos` plugin for service discovery.

```go
// server/main.go
package main

import (
	"flag"
	"log"

	"github.com/marsevilspirit/phobos/example"
	"github.com/marsevilspirit/phobos/server"
	"github.com/marsevilspirit/phobos/serverplugin"
)

var (
	addr     = flag.String("addr", "localhost:8972", "server address")
	deimosAddr = flag.String("deimosAddr", "localhost:4001", "deimos address")
	basePath = flag.String("basePath", "/phobos_services", "base path for deimos")
)

func main() {
	flag.Parse()

	s := server.NewServer()
	
	// Use the Deimos plugin
	plugin := serverplugin.NewDeimosPlugin(*deimosAddr, *basePath, s, 0)
	err := plugin.Start()
	if err != nil {
		log.Fatal(err)
	}
	s.Plugins.Add(plugin)

	s.RegisterWithName("HelloWorld", new(example.HelloWorld), "")
	s.Serve("tcp", *addr)
}
```

### 4. Create the Phobos Client

Create a client that discovers the `HelloWorld` service through Deimos.

```go
// client/main.go
package main

import (
	"context"
	"flag"
	"log"

	"github.com/marsevilspirit/phobos/client"
	"github.com/marsevilspirit/phobos/example"
)

var (
	deimosAddr = flag.String("deimosAddr", "localhost:4001", "deimos address")
	basePath = flag.String("basePath", "/phobos_services", "base path for deimos")
)

func main() {
	flag.Parse()

	// Discover services via Deimos
	d := client.NewDeimosDiscovery(*basePath, []string{*deimosAddr}, nil)
	xclient := client.NewXClient("HelloWorld", client.Failtry, client.RandomSelect, d, client.DefaultOption)
	defer xclient.Close()

	args := &example.Args{
		First: "Mars",
	}

	reply := &example.Reply{}

	err := xclient.Call(context.Background(), "Greet", args, reply)
	if err != nil {
		log.Fatalf("failed to call: %v", err)
	}

	log.Printf("reply: %s", reply.Second)
}
```
