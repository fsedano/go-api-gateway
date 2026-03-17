# Simple Go API Gateway

A minimal API gateway written in Go using `net/http/httputil.ReverseProxy`.

## Getting started

1. Start the example backend service:

```sh
go run ./backend
```

2. Run the gateway:

```sh
API_GATEWAY_BACKEND=http://localhost:8081 go run .
```

3. Send requests through the gateway:

```sh
curl http://localhost:8080/hello
```

## Features

- Reverse proxy to upstream backend
- `/health` endpoint
- Adds `X-API-Gateway: simple-go-gateway`

## Customization

Update `proxyDirector` in `main.go` to implement header manipulation, routing, auth checks, or other request transformations.
