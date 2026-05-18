package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"yadro.com/course/closers"
	updatepb "yadro.com/course/proto/update"
	"yadro.com/course/update/adapters/broker"
	"yadro.com/course/update/adapters/db"
	updategrpc "yadro.com/course/update/adapters/grpc"
	"yadro.com/course/update/adapters/words"
	"yadro.com/course/update/adapters/xkcd"
	"yadro.com/course/update/config"
	"yadro.com/course/update/core"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "server configuration file")
	flag.Parse()

	cfg := config.MustLoad(configPath)

	logger := mustMakeLogger(cfg.LogLevel)

	err := run(cfg, logger)
	if err != nil {
		logger.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func run(cfg config.Config, log *slog.Logger) error {
	log.Info("starting server")
	log.Debug("debug messages are enabled")

	storage, err := db.New(log, cfg.DBAddress)
	if err != nil {
		return fmt.Errorf("failed to connect to db: %v", err)
	}
	defer closers.CloseOrLog(storage, log)

	migrateErr := storage.Migrate()
	if migrateErr != nil {
		return fmt.Errorf("failed to migrate db: %v", migrateErr)
	}

	xkcdClient, err := xkcd.NewClient(cfg.XKCD.URL, cfg.XKCD.Timeout, log)
	if err != nil {
		return fmt.Errorf("failed create XKCD client: %v", err)
	}

	wordsClient, err := words.NewClient(cfg.WordsAddress, log)
	if err != nil {
		return fmt.Errorf("failed create Words client: %v", err)
	}
	defer closers.CloseOrLog(wordsClient, log)

	eventPublisher, err := broker.NewPublisher(cfg.BrokerAddress, log)
	if err != nil {
		return fmt.Errorf("failed to connect to broker: %v", err)
	}
	defer closers.CloseOrLog(eventPublisher, log)

	updateService, err := core.NewService(log, storage, xkcdClient, wordsClient, eventPublisher, cfg.XKCD.Concurrency)
	if err != nil {
		return fmt.Errorf("failed create Update service: %v", err)
	}

	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	updateServer := updategrpc.NewServer(updateService)
	updatepb.RegisterUpdateServer(grpcServer, updateServer)
	reflection.Register(grpcServer)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		<-ctx.Done()
		log.Debug("shutting down server")
		grpcServer.GracefulStop()
	}()

	serveErr := grpcServer.Serve(listener)
	if serveErr != nil {
		return fmt.Errorf("failed to serve: %v", serveErr)
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
