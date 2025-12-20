package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	wordspb "yadro.com/course/proto/words"
	"yadro.com/course/words/words"
)

const (
	maxPhraseLen    = 4096 * 100
	maxShutdownTime = 5 * time.Second
)

type server struct {
	wordspb.UnimplementedWordsServer
}

func (s *server) Ping(_ context.Context, in *emptypb.Empty) (*emptypb.Empty, error) {
	return nil, nil
}

func (s *server) Norm(_ context.Context, in *wordspb.WordsRequest) (*wordspb.WordsReply, error) {
	if len(in.GetPhrase()) > maxPhraseLen {
		return nil, status.Error(
			codes.ResourceExhausted,
			"phrase is large than "+strconv.Itoa(maxPhraseLen),
		)
	}
	return &wordspb.WordsReply{
		Words: words.Norm(in.GetPhrase()),
	}, nil
}

type Config struct {
	Port string `yaml:"port" env:"WORDS_GRPC_PORT" env-default:"8080"`
}

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "server configuration file")
	flag.Parse()

	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		panic(err)
	}

	log := slog.New(slog.NewTextHandler(
		os.Stdout,
		&slog.HandlerOptions{
			Level:     slog.LevelDebug,
			AddSource: true,
		},
	))

	if err := run(cfg, log); err != nil {
		log.Error("failed to run", "error", err)
		os.Exit(1)
	}
}

// overcomplicated code for very simple grpc server
func run(cfg Config, log *slog.Logger) error {
	listener, err := net.Listen("tcp", ":"+cfg.Port)
	if err != nil {
		return fmt.Errorf("failed to listen port %s: %v", cfg.Port, err)
	}

	s := grpc.NewServer()
	wordspb.RegisterWordsServer(s, &server{})
	reflection.Register(s)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go func() {
		log.Info("starting server", "port", cfg.Port)
		if err = s.Serve(listener); err != nil {
			log.Error("faied to serve", "error", err)
			// notify shutdown code if needed
			cancel()
		}
	}()

	// got failed serve or terminate signal
	<-ctx.Done()

	// force after some time
	timer := time.AfterFunc(maxShutdownTime, func() {
		log.Info("forcing server stop")
		s.Stop()
	})
	defer timer.Stop()

	log.Info("starting graceful stop")
	s.GracefulStop()
	log.Info("server stopped")

	return err
}
