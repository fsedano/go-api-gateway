package main

import (
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"
)

// Simple reverse-proxy based API gateway.
//
// Usage:
//   API_GATEWAY_BACKEND=http://localhost:8081 go run .
//
// This gateway forwards all requests to the configured backend.
// Customize in `proxyDirector` for header manipulation, routing, etc.

func main() {
	backend := os.Getenv("API_GATEWAY_BACKEND")
	if backend == "" {
		log.Fatal("API_GATEWAY_BACKEND environment variable must be set (e.g. http://localhost:8081)")
	}

	backendURL, err := url.Parse(backend)
	if err != nil {
		log.Fatalf("invalid backend URL %q: %v", backend, err)
	}

	proxy := httputil.NewSingleHostReverseProxy(backendURL)
	proxy.Director = proxyDirector(backendURL)

	// Tune the transport for higher throughput and connection reuse.
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	proxy.Transport = transport

	client := &http.Client{Transport: transport}

	proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
		log.Printf("proxy error: %v", err)
		rw.WriteHeader(http.StatusBadGateway)
		rw.Write([]byte("Bad Gateway"))
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{\"status\":\"ok\"}"))
	})

	// Ensure the runtime uses all available cores.
	runtime.GOMAXPROCS(runtime.NumCPU())

	logRequests := strings.EqualFold(os.Getenv("API_GATEWAY_LOG"), "true")
	useDirectProxy := strings.EqualFold(os.Getenv("API_GATEWAY_MODE"), "direct")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if logRequests {
			log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL)
		}
		if useDirectProxy {
			directProxy(w, r, backendURL, client)
			return
		}
		proxy.ServeHTTP(w, r)
	})

	addr := ":8080"
	if port := os.Getenv("PORT"); port != "" {
		addr = ":" + port
	}

	server := &http.Server{
		Addr:         addr,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("starting api gateway on %s -> backend %s", addr, backendURL)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}

func proxyDirector(backendURL *url.URL) func(*http.Request) {
	return func(req *http.Request) {
		// Preserve the original request path and query.
		req.URL.Scheme = backendURL.Scheme
		req.URL.Host = backendURL.Host
		// Keep the original path as-is; modify here for routing rules.

		// Forward Host header by default.
		req.Host = backendURL.Host

		// Example: Add a header to indicate the request came through the gateway.
		req.Header.Set("X-API-Gateway", "simple-go-gateway")

		// Example: strip /api prefix
		if strings.HasPrefix(req.URL.Path, "/api") {
			req.URL.Path = strings.TrimPrefix(req.URL.Path, "/api")
			if req.URL.Path == "" {
				req.URL.Path = "/"
			}
		}
	}
}

func directProxy(w http.ResponseWriter, r *http.Request, backendURL *url.URL, client *http.Client) {
	// Build target URL from incoming request.
	targetURL := *r.URL
	targetURL.Scheme = backendURL.Scheme
	targetURL.Host = backendURL.Host

	outReq, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL.String(), r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("Bad Gateway"))
		return
	}

	// Copy headers
	outReq.Header = r.Header.Clone()
	outReq.Host = backendURL.Host

	resp, err := client.Do(outReq)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("Bad Gateway"))
		return
	}
	defer resp.Body.Close()

	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}
