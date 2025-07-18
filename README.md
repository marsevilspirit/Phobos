# Phobos: A Feature-Rich RPC Framework for Service Governance

Phobos is a powerful and extensible RPC framework designed for building robust and scalable microservices. It is inspired by the source code of `net/rpc` and extends it with a wide range of features for service governance.

## Features

Phobos offers a comprehensive set of features to simplify the development, deployment, and management of microservices.

### Core Features

*   **High-Performance RPC:** Built on a lightweight and efficient core, Phobos provides low-latency, high-throughput communication between services.
*   **Multiple Serialization Protocols:** Supports various serialization formats, including JSON, Msgpack, and Protocol Buffers, allowing for flexibility and performance optimization.
*   **Gzip Compression:** Automatically compresses large data payloads using Gzip to reduce network bandwidth consumption.
*   **HTTP Gateway:** A built-in gateway allows for seamless conversion between HTTP and RPC protocols, enabling web clients to interact with Phobos services.
*   **Timeout Management:** Provides fine-grained control over request timeouts to prevent cascading failures and ensure service responsiveness.
*   **Flexible Metadata:** Leverages metadata to pass contextual information between services, enabling advanced features like distributed tracing and authentication.

### Service Governance

*   **Service Registration and Discovery:** Integrates with [Deimos](https://github.com/marsevilspirit/Deimos) for dynamic service registration and discovery, allowing services to locate and communicate with each other without hardcoded addresses.
*   **Load Balancing:** Supports multiple load balancing strategies, including:
    *   **Random:** Distributes requests randomly among available servers.
    *   **Round Robin:** Distributes requests in a round-robin fashion.
    *   **Weighted Round Robin:** Distributes requests based on server weights.
    *   **Consistent Hash:** Ensures that requests for the same key are routed to the same server.
    *   **Closest:** Selects the server with the lowest latency.
*   **Circuit Breaker:** Implements a circuit breaker pattern to prevent a service from repeatedly trying to connect to a failing service, improving overall system resilience.
*   **Heartbeat:** Sends periodic heartbeat messages to monitor the health of servers and detect failures quickly.
*   **Metrics and Monitoring:** Integrates with Prometheus for collecting and exposing metrics, and Grafana for visualizing them, providing insights into service performance and health.

## Concepts

Phobos is composed of three main components:

*   **Server:** The core of the framework, responsible for registering and exposing services.
*   **Client:** The client-side library that enables services to consume other services. It provides features like service discovery, load balancing, and failure handling.
*   **Gateway:** An optional component that acts as an HTTP gateway to Phobos services, allowing them to be accessed by web clients.

## Installation

To use Phobos in your project, you can use `go get`:

```bash
go get github.com/marsevilspirit/phobos
```

## Usage

Here's a basic example of how to use Phobos to create a simple "Hello, World" service.

### 1. Define the Service

First, define the service interface and its implementation:

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

### 2. Create the Server

Next, create a server to host the service:

```go
package main

import (
	"flag"

	"github.com/marsevilspirit/phobos/example"
	"github.com/marsevilspirit/phobos/server"
)

var addr = flag.String("addr", "localhost:8972", "server address")

func main() {
	flag.Parse()

	s := server.NewServer(nil)
	s.RegisterWithName("HelloWorld", new(example.HelloWorld), "")
	s.Serve("tcp", *addr)
}
```

### 3. Create the Client

Finally, create a client to consume the service:

```go
package main

import (
	"context"
	"flag"
	"log"

	"github.com/marsevilspirit/phobos/client"
	"github.com/marsevilspirit/phobos/example"
)

var addr = flag.String("addr", "localhost:8972", "server address")

func main() {
	flag.Parse()

	d := client.NewP2PDiscovery("tcp@"+*addr, "")
	xclient := client.NewXClient("HelloWorld", client.Failtry, client.RandomSelect, d, client.DefaultOption)
	defer xclient.Close()

	args := &example.Args{
		First: "budei",
	}

	reply := &example.Reply{}

	err := xclient.Call(context.Background(), "Greet", args, reply)
	if err != nil {
		log.Fatalf("failed to call: %v", err)
	}

	log.Print("reply: ", reply.Second)
}
```

This is just a basic example. For more advanced usage, including service discovery, load balancing, and other features, please refer to the examples in the `example` directory.