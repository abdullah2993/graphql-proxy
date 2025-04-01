# GraphQL Proxy

A lightweight, configurable GraphQL proxy server that supports operation-based routing and load balancing.

## Features

- Operation-based routing (query/mutation/subscription)
- Operation name-based routing
- Weighted load balancing
- Local metrics tracking
- Configurable timeouts and connection settings
- JSON/Text logging with configurable output
- Support for both GET and POST requests
- GraphQL operation validation
- Header forwarding

## Installation

```bash
go install github.com/abdullah2993/graphql-proxy/cmd/gqlproxy@latest
```

## Usage

1. Create a config file (default: `config.yaml`):

```yaml
server:
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 90s
  max_idle_conns: 100
  max_idle_conns_host: 10
  handshake_timeout: 10s
  response_timeout: 30s

logging:
  level: info
  format: json
  output: stdout

upstreams:
  - url: "http://graphql1:8080/graphql"
    capabilities:
      - query
      - mutation
    operation_names:
      - getUserProfile
      - updateUserProfile
    weight: 2
```

2. Run the proxy:

```bash
gqlproxy -addr :8080 -config config.yaml
```

## Configuration

### Server Settings

- `read_timeout`: Maximum duration for reading the entire request
- `write_timeout`: Maximum duration for writing the response
- `idle_timeout`: Maximum duration to wait for the next request
- `max_idle_conns`: Maximum number of idle connections
- `max_idle_conns_host`: Maximum idle connections per host
- `handshake_timeout`: Maximum duration for TLS handshake
- `response_timeout`: Maximum duration for upstream response

### Logging Settings

- `level`: Log level (debug, info, warn, error)
- `format`: Log format (json, text)
- `output`: Log output (stdout, stderr, or file path)

### Upstream Settings

- `url`: GraphQL server endpoint
- `capabilities`: List of supported operations (query, mutation, subscription)
- `operation_names`: List of operation names this server can handle (optional)
- `weight`: Load balancing weight (higher number = more traffic)

## API

The proxy accepts GraphQL requests via:

### POST
- Content-Type: application/json
```json
{
  "query": "query { ... }",
  "variables": {},
  "operationName": "optional"
}
```
- Content-Type: application/graphql
```graphql
query {
  ...
}
```

### GET
```
/graphql?query=query{...}&variables={}&operationName=optional
```

## Load Balancing

The proxy uses weighted random selection to distribute requests among eligible upstream servers. A server is considered eligible if it:
1. Supports the operation type (query/mutation/subscription)
2. Can handle the specific operation name (if configured)
