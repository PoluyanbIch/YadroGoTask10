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

	searchpb "yadro.com/course/proto/search"
	"yadro.com/course/search/adapters/db"
	searchgrpc "yadro.com/course/search/adapters/grpc"
	"yadro.com/course/search/adapters/initiator"
	"yadro.com/course/search/adapters/nats"
	"yadro.com/course/search/adapters/words"
	"yadro.com/course/search/core"

	"yadro.com/course/search/config"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "server configuration file")
	flag.Parse()
	cfg := config.MustLoad(configPath)

	log := mustMakeLogger(cfg.LogLevel)

	if err := run(cfg, log); err != nil {
		log.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func run(cfg config.Config, log *slog.Logger) error {
	log.Info("starting server")
	log.Debug("debug messages are enabled")

	// database adapter
	storage, err := db.New(log, cfg.DBAddress)
	if err != nil {
		return fmt.Errorf("failed to connect to db: %v", err)
	}
	if err := storage.Migrate(); err != nil {
		return fmt.Errorf("failed to migrate db: %v", err)
	}

	// words adapter
	words, err := words.NewClient(cfg.WordsAddress, log)
	if err != nil {
		return fmt.Errorf("failed create Words client: %v", err)
	}

	// nats adapter
	subscriber, err := nats.NewNatsSubscriber(log, cfg.BrokerAddress)
	if err != nil {
		return fmt.Errorf("failed create Nats adapter")
	}

	// service
	searcher := core.NewService(log, storage, words)

	// context for Ctrl-C
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// NATS subscriptions
	err = subscriber.Subscribe(ctx, "xkcd.db.update", func(ctx context.Context, _ []byte) error {
		return searcher.BuildIndex(ctx)
	})
	if err != nil {
		return fmt.Errorf("failed subscribe udpate: %v", err)
	}
	err = subscriber.Subscribe(ctx, "xkcd.db.drop", func(ctx context.Context, _ []byte) error {
		return searcher.BuildIndex(ctx)
	})
	if err != nil {
		return fmt.Errorf("failed subscribe drop: %v", err)
	}

	// initiator
	init := initiator.NewInitiator(cfg.IndexTTL, searcher, log)
	init.Start(ctx)

	// grpc server
	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	searchpb.RegisterSearchServer(s, searchgrpc.NewServer(searcher))
	reflection.Register(s)

	go func() {
		<-ctx.Done()
		log.Debug("shutting down server")
		s.GracefulStop()
	}()

	if err := s.Serve(listener); err != nil {
		return fmt.Errorf("failed to serve: %v", err)
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
		panic("unknown log level + " + logLevel)
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	return slog.New(handler)
}
