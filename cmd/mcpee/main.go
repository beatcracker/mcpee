package main

import (
	"context"
	"flag"
	"log/slog"
	"mcpee"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var defaultConfigFile = "mcpee.yaml"

func main() {
	configPath := flag.String("config", defaultConfigFile, "path to configuration file")
	flag.Parse()
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg, err := mcpee.LoadConfig(*configPath)
	if err != nil {
		log.Error("load config failed", "error", err)
		os.Exit(1)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	backends := make([]mcpee.Backend, 0, len(cfg.Backends))
	for _, bc := range cfg.Backends {
		b, err := mcpee.ConnectStdioBackend(ctx, bc)
		if err != nil {
			_ = mcpee.CloseBackends(backends)
			log.Error("backend startup failed", "backend", bc.Name, "error", err)
			os.Exit(1)
		}
		backends = append(backends, b)
		log.Info(
			"backend connected",
			"backend", b.Name(),
			"cwd", bc.Cwd,
			"inherit_env", bc.InheritEnv,
			"max_frame_bytes", bc.EffectiveMaxFrameBytes(),
		)
		log.Info("backend catalog loaded", "backend", b.Name(), "tools", len(b.Catalog()))
	}
	defer mcpee.CloseBackends(backends)
	catalog, err := mcpee.ProjectCatalog(backends)
	if err != nil {
		log.Error("catalog projection failed", "error", err)
		os.Exit(1)
	}
	server := mcpee.NewMCPServer(catalog, log)
	httpServer := &http.Server{Addr: cfg.Listen, Handler: mcpee.HTTPHandler(server)}
	go func() {
		<-ctx.Done()
		log.Info("shutdown")
		sdctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(sdctx)
	}()
	log.Info("server listening", "address", cfg.Listen, "endpoint", "/mcp", "tools", len(catalog.Tools()))
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error("server failed", "error", err)
		os.Exit(1)
	}
}
