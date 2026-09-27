package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"nodo/external/openrouter"
	"nodo/internal/auth"
	"nodo/internal/config"
	"nodo/internal/database"
	"nodo/internal/quiz"
	"nodo/internal/server"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}
	ctx := context.Background()
	db, err := database.Open(ctx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns, cfg.DBMaxLifetime, cfg.DBMaxIdleTime)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	if err := database.Migrate(ctx, db); err != nil {
		logger.Error("database migration failed", "error", err)
		os.Exit(1)
	}
	authService := auth.NewService(db, cfg.SessionTTL, auth.NewJWT(cfg.JWTSecret))
	openRouterClient := openrouter.New(cfg.OpenRouterAPIKey, cfg.OpenRouterModel, &http.Client{Timeout: 25 * time.Second})
	quizService := quiz.NewService(quiz.NewLLMGenerator(openRouterClient))
	handler := server.New(auth.NewHTTP(authService), quiz.NewHTTP(quizService, logger), logger, cfg.FrontendOrigin, func() error { return db.Ping(context.Background()) })
	httpServer := &http.Server{Addr: cfg.HTTPAddr, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 35 * time.Second, WriteTimeout: 35 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		logger.Info("server started", "address", cfg.HTTPAddr, "environment", cfg.Environment)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()
	shutdown, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-shutdown.Done()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
}
