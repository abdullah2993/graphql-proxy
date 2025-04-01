package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/abdullah2993/graphql-proxy/pkgs/config"
	"github.com/abdullah2993/graphql-proxy/pkgs/graphql"
	"github.com/abdullah2993/graphql-proxy/pkgs/loadbalancer"
	"github.com/abdullah2993/graphql-proxy/pkgs/metrics"
)

type Proxy struct {
	lb      *loadbalancer.LoadBalancer
	logger  *slog.Logger
	client  *http.Client
	metrics *metrics.Metrics
}

func NewProxy(cfg *config.Config, logger *slog.Logger) *Proxy {
	return &Proxy{
		lb:     loadbalancer.New(cfg.Upstreams),
		logger: logger,
		client: &http.Client{
			Timeout: cfg.Server.ResponseTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        cfg.Server.MaxIdleConns,
				MaxIdleConnsPerHost: cfg.Server.MaxIdleConnsHost,
				IdleConnTimeout:     cfg.Server.IdleTimeout,
				TLSHandshakeTimeout: cfg.Server.HandshakeTimeout,
			},
		},
		metrics: metrics.New(),
	}
}

func (p *Proxy) Handler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	p.metrics.IncActiveRequests()

	requestID := fmt.Sprintf("%d", time.Now().UnixNano())
	ctx := r.Context()
	logger := p.logger.With(
		"method", r.Method,
		"path", r.URL.Path,
		"remote_addr", r.RemoteAddr,
		"request_id", requestID,
	)

	req, err := graphql.ParseGraphQLRequest(r)
	if err != nil {
		logger.ErrorContext(ctx, "failed to parse GraphQL request", "error", err)
		http.Error(w, fmt.Sprintf("Bad Request: %v", err), http.StatusBadRequest)
		return
	}

	op, name, err := req.Parse()
	if err != nil {
		logger.ErrorContext(ctx, "failed to parse GraphQL operation", "error", err)
		http.Error(w, fmt.Sprintf("Invalid GraphQL query: %v", err), http.StatusBadRequest)
		return
	}
	defer func() {
		p.metrics.DecActiveRequests()
		duration := time.Since(start)
		success := err == nil
		p.metrics.RecordRequest(string(op), duration, success)
	}()

	logger = logger.With(
		"operation", op,
		"operation_name", name,
	)

	server, err := p.lb.GetServer(config.Capability(op), name)
	if err != nil {
		logger.ErrorContext(ctx, "no server available", "error", err)
		http.Error(w, fmt.Sprintf("No server available for operation: %v", err), http.StatusServiceUnavailable)
		return
	}

	logger = logger.With("upstream", server.URL)

	// Marshal the request body
	body, err := json.Marshal(req)
	if err != nil {
		logger.ErrorContext(ctx, "failed to marshal request body", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Create upstream request (always POST)
	upstreamReq, err := http.NewRequestWithContext(ctx, http.MethodPost, server.URL, bytes.NewReader(body))
	if err != nil {
		logger.ErrorContext(ctx, "failed to create upstream request", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Set headers
	upstreamReq.Header.Set("Content-Type", "application/json")
	upstreamReq.Header.Set("X-Forwarded-Host", r.Host)
	upstreamReq.Header.Set("X-Forwarded-Proto", r.URL.Scheme)
	upstreamReq.Header.Set("X-Forwarded-For", r.RemoteAddr)
	upstreamReq.Header.Set("X-Request-ID", requestID)

	// Copy original headers (except those we explicitly set)
	for k, vv := range r.Header {
		if k != "Content-Type" && !isForwardedHeader(k) {
			for _, v := range vv {
				upstreamReq.Header.Add(k, v)
			}
		}
	}
	upstreamStart := time.Now()

	// Send request
	resp, err := p.client.Do(upstreamReq)
	if err != nil {
		logger.ErrorContext(ctx, "failed to send request to upstream", "error", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	defer func() {
		latency := time.Since(upstreamStart)
		success := err == nil && resp.StatusCode < 500
		p.metrics.RecordUpstreamRequest(server.URL, latency, success)
	}()

	// Copy response headers
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}

	// Copy status code
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	if _, err := io.Copy(w, resp.Body); err != nil {
		logger.ErrorContext(ctx, "error copying response", "error", err)
		return
	}

	logger.InfoContext(ctx, "proxied operation",
		"status_code", resp.StatusCode,
		"content_length", resp.ContentLength,
	)
}

func (p *Proxy) MetricsHandler(w http.ResponseWriter, r *http.Request) {
	stats := p.metrics.GetStats()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func isForwardedHeader(header string) bool {
	switch header {
	case "X-Forwarded-Host", "X-Forwarded-Proto", "X-Forwarded-For", "X-Request-ID":
		return true
	default:
		return false
	}
}
