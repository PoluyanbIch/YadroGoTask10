package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"

	"yadro.com/course/api/adapters/aaa"
	"yadro.com/course/api/adapters/rest"
	"yadro.com/course/api/adapters/rest/middleware"
	"yadro.com/course/api/adapters/search"
	"yadro.com/course/api/adapters/update"
	"yadro.com/course/api/adapters/words"
	"yadro.com/course/api/config"
	"yadro.com/course/api/core"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "server configuration file")
	flag.Parse()

	cfg := config.MustLoad(configPath)

	log := mustMakeLogger(cfg.LogLevel)

	log.Info("starting server")
	log.Debug("debug messages are enabled")

	updateClient, err := update.NewClient(cfg.UpdateAddress, log)
	if err != nil {
		log.Error("cannot init words adapter", "error", err)
		os.Exit(1)
	}

	wordsClient, err := words.NewClient(cfg.WordsAddress, log)
	if err != nil {
		log.Error("cannot init words adapter", "error", err)
		os.Exit(1)
	}

	searchClient, err := search.NewClient(cfg.SearchAddress, log)
	if err != nil {
		log.Error("cannot init search adapter", "error", err)
		os.Exit(1)
	}

	aaaClient, err := aaa.NewClient(cfg.AAAAddress, log)
	if err != nil {
		log.Error("cannot init aaa adapter", "error", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()

	mux.Handle("POST /auth/login", rest.NewLoginHandler(log, aaaClient, cfg))
	mux.Handle("POST /auth/register", rest.NewRegisterHandler(log, aaaClient, cfg))
	mux.Handle("POST /auth/refresh", rest.NewRefreshHandler(log, aaaClient, cfg))
	mux.Handle("POST /auth/logout", rest.NewLogoutHandler(log))

	mux.Handle("GET /api/recent_views", middleware.RequiredAuthMiddleware(rest.NewGetRecentViewsHandler(log, aaaClient, cfg), aaaClient))
	mux.Handle("GET /api/recent_searches", middleware.RequiredAuthMiddleware(rest.NewGetRecentSearchesHandler(log, aaaClient, cfg), aaaClient))

	mux.Handle("GET /api/ping", rest.NewPingHandler(log, map[string]core.Pinger{"words": wordsClient, "update": updateClient, "search": searchClient, "aaa": aaaClient}, cfg))

	mux.Handle("GET /api/words", rest.NewWordsHandler(log, wordsClient, cfg))

	mux.Handle("GET /api/search",
		middleware.Concurrency(
			middleware.OptionalAuthMiddleware( // 👈 Проверяет токен если есть
				middleware.WithSearchTracking( // 👈 Трекает только если userID в контексте
					rest.NewSearchHandler(log, searchClient, cfg),
					aaaClient,
					log,
				),
				aaaClient,
			),
			cfg.SearchConcurrency,
		),
	)
	mux.Handle("GET /api/isearch",
		middleware.Rate(
			middleware.OptionalAuthMiddleware( // 👈 Проверяет токен если есть
				middleware.WithSearchTracking( // 👈 Трекает только если userID в контексте
					rest.NewSearchHandler(log, searchClient, cfg),
					aaaClient,
					log,
				),
				aaaClient,
			),
			cfg.SearchRate,
		),
	)

	mux.Handle("GET /api/comic", middleware.OptionalAuthMiddleware(middleware.WithComicViewTracking(rest.NewGetComicHandler(log, searchClient, cfg), aaaClient, log), aaaClient))
	mux.Handle("GET /api/recommendations", middleware.RequiredAuthMiddleware(rest.NewGetRecommendationsHandler(log, searchClient, aaaClient, cfg), aaaClient))
	mux.Handle("GET /api/latest_comics", rest.NewGetLatestComicsHandler(log, searchClient, cfg))

	mux.Handle("POST /api/db/update", middleware.AdminMiddleware(rest.NewUpdateHandler(log, updateClient, cfg), aaaClient))
	mux.Handle("GET /api/db/stats", rest.NewUpdateStatsHandler(log, updateClient, cfg))
	mux.Handle("GET /api/db/status", rest.NewUpdateStatusHandler(log, updateClient, cfg))
	mux.Handle("DELETE /api/db", middleware.AdminMiddleware(rest.NewDropHandler(log, updateClient, cfg), aaaClient))

	h := middleware.CorsMiddleware(mux, "http://localhost:3000,http://frontend:80")

	server := http.Server{
		Addr:        cfg.HTTPConfig.Address,
		ReadTimeout: cfg.HTTPConfig.Timeout,
		Handler:     h,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		<-ctx.Done()
		log.Debug("shutting down server")
		if err := server.Shutdown(context.Background()); err != nil {
			log.Error("erroneous shutdown", "error", err)
		}
	}()

	log.Info("Running HTTP server", "address", cfg.HTTPConfig.Address)
	if err := server.ListenAndServe(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			log.Error("server closed unexpectedly", "error", err)
			return
		}
	}
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
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	return slog.New(handler)
}
