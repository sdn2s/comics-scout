package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"

	aaa "yadro.com/course/api/adapters/aaa"
	"yadro.com/course/api/adapters/rest"
	"yadro.com/course/api/adapters/rest/middleware"
	searchadapter "yadro.com/course/api/adapters/search"
	"yadro.com/course/api/adapters/update"
	"yadro.com/course/api/adapters/words"
	"yadro.com/course/api/config"
	"yadro.com/course/api/core"
	"yadro.com/course/closers"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "server configuration file")
	flag.Parse()

	cfg := config.MustLoad(configPath)

	log := mustMakeLogger(cfg.LogLevel)

	if err := run(cfg, log); err != nil {
		slog.Error("run failed", "error", err)
		os.Exit(1)
	}
}

func run(cfg config.Config, log *slog.Logger) error {
	log.Info("starting server")
	log.Debug("debug messages are enabled")

	wordsClient, err := words.NewClient(cfg.WordsAddress, log)
	if err != nil {
		return fmt.Errorf("cannot init words adapter: %w", err)
	}
	defer closers.CloseOrLog(wordsClient, log)

	updateClient, err := update.NewClient(cfg.UpdateAddress, log)
	if err != nil {
		return fmt.Errorf("cannot init update adapter: %w", err)
	}
	defer closers.CloseOrLog(updateClient, log)

	searchClient, err := searchadapter.NewClient(cfg.SearchAddress, log)
	if err != nil {
		return fmt.Errorf("cannot init search adapter: %w", err)
	}
	defer closers.CloseOrLog(searchClient, log)

	auth, err := aaa.New(cfg.TokenTTL, log)
	if err != nil {
		return fmt.Errorf("cannot init aaa: %w", err)
	}

	mux := http.NewServeMux()

	mux.Handle("POST /api/login", rest.NewLoginHandler(log, auth))

	mux.Handle("POST /api/db/update",
		middleware.Auth(rest.NewUpdateHandler(log, updateClient), auth),
	)
	mux.Handle("DELETE /api/db",
		middleware.Auth(rest.NewDropHandler(log, updateClient), auth),
	)

	mux.Handle("GET /api/db/stats", rest.NewUpdateStatsHandler(log, updateClient))
	mux.Handle("GET /api/db/status", rest.NewUpdateStatusHandler(log, updateClient))

	searchHandler := rest.NewSearchHandler(log, searchClient)
	mux.Handle("GET /api/search",
		middleware.Concurrency(searchHandler.ServeHTTP, cfg.SearchConcurrency),
	)

	indexSearchHandler := rest.NewIndexSearchHandler(log, searchClient)
	mux.Handle("GET /api/isearch",
		middleware.Rate(indexSearchHandler.ServeHTTP, cfg.SearchRate),
	)

	deps := map[string]core.Pinger{
		"words":  wordsClient,
		"update": updateClient,
		"search": searchClient,
	}
	mux.Handle("GET /api/ping", rest.NewPingHandler(log, deps))

	server := http.Server{
		Addr:        cfg.HTTPConfig.Address,
		ReadTimeout: cfg.HTTPConfig.Timeout,
		Handler:     mux,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		<-ctx.Done()
		log.Debug("shutting down server")

		if shutdownErr := server.Shutdown(context.Background()); shutdownErr != nil {
			log.Error("erroneous shutdown", "error", shutdownErr)
		}
	}()

	log.Info("Running HTTP server", "address", cfg.HTTPConfig.Address)

	if err := server.ListenAndServe(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("server closed unexpectedly: %w", err)
		}
	}

	return nil
}

func mustMakeLogger(logLevel string) *slog.Logger {
	var level slog.Level

	switch logLevel {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "ERROR":
		level = slog.LevelError
	default:
		panic("unknown log level: " + logLevel)
	}

	handlerOptions := &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	}

	handler := slog.NewTextHandler(os.Stderr, handlerOptions)
	logger := slog.New(handler)

	return logger
}
