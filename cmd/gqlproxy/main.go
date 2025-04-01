package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/abdullah2993/graphql-proxy/pkgs/config"
	"github.com/abdullah2993/graphql-proxy/pkgs/logging"
	"github.com/abdullah2993/graphql-proxy/pkgs/proxy"
)

var (
	addr       = flag.String("addr", ":8080", "http service address")
	configFile = flag.String("config", "config.yaml", "path to config file")
)

func main() {
	flag.Parse()

	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger := logging.NewLogger(cfg.Logging.Level, cfg.Logging.Format, cfg.Logging.Output)
	slog.SetDefault(logger)

	logger.Info("starting GraphQL proxy server", "address", *addr)

	proxy := proxy.NewProxy(cfg, logger)
	server := &http.Server{
		Addr:    *addr,
		Handler: http.HandlerFunc(proxy.Handler),
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", http.HandlerFunc(proxy.MetricsHandler))
	mux.Handle("/v1/graphql", http.HandlerFunc(proxy.Handler))
	server.Handler = mux

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigChan
		logger.Info("received shutdown signal", "signal", sig)

		if err := server.Shutdown(context.Background()); err != nil {
			logger.Error("server shutdown error", "error", err)
		}
	}()

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}
