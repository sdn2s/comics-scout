package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "path to config file")
	flag.Parse()

	cfg := mustLoadConfig(configPath)
	logger := makeLogger(cfg.LogLevel)

	templates, err := parseTemplates()
	if err != nil {
		logger.Error("failed to parse templates", "error", err)
		os.Exit(1)
	}

	api := NewAPIClient(cfg.APIBaseURL, cfg.RequestTimeout)
	srv := newServer(cfg, logger, api, templates)

	server := &http.Server{
		Addr:              cfg.Address,
		Handler:           srv.routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		<-ctx.Done()
		logger.Info("shutting down web ui")
		if err := server.Shutdown(context.Background()); err != nil {
			logger.Error("web ui shutdown failed", "error", err)
		}
	}()

	logger.Info("web ui is running", "address", cfg.Address, "api", cfg.APIBaseURL)
	if err := server.ListenAndServe(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("web ui stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	}
}

func makeLogger(level string) *slog.Logger {
	var slogLevel slog.Level

	switch strings.ToUpper(level) {
	case "DEBUG":
		slogLevel = slog.LevelDebug
	case "ERROR":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level:     slogLevel,
		AddSource: true,
	})

	return slog.New(handler)
}
