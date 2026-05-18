package main

import (
	"flag"
	"log/slog"
	"net"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	searchpb "yadro.com/course/proto/search"
	"yadro.com/course/search/config"
	"yadro.com/course/search/service"
	"yadro.com/course/update/adapters/words"
)

func main() {
	var cfgPath string
	flag.StringVar(&cfgPath, "config", "config.yaml", "server configuration file")
	flag.Parse()

	cfg := config.MustLoad(cfgPath)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if err := run(cfg, logger); err != nil {
		logger.Error("search service failed", "error", err)
		os.Exit(1)
	}
}

func run(cfg config.Config, logger *slog.Logger) error {
	db, err := sqlx.Connect("pgx", cfg.DBAddress)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := db.Close(); cerr != nil {
			logger.Error("failed to close db", "error", cerr)
		}
	}()

	wordsClient, err := words.NewClient(cfg.WordsTarget, logger)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := wordsClient.Close(); cerr != nil {
			logger.Error("failed to close words client", "error", cerr)
		}
	}()

	svc := service.New(logger, db, wordsClient)

	lis, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	searchpb.RegisterSearchServer(grpcServer, &server{service: svc})
	reflection.Register(grpcServer)

	logger.Info("search service listening", "addr", cfg.Address)
	if err := grpcServer.Serve(lis); err != nil {
		return err
	}

	return nil
}
