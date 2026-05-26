package main

import (
	"log/slog"
	"os"

	"github.com/ivanSaichkin/wb-search-top/internal/app/factory"
	"github.com/ivanSaichkin/wb-search-top/internal/config"
	"github.com/ivanSaichkin/wb-search-top/internal/logger"
)

func main() {
	cfg := config.Load()
	log := logger.InitLogger(cfg.Logger)
	slog.SetDefault(log)

	slog.Info("Logger initialized", "level", cfg.Logger.Level, "format", cfg.Logger.Format)

	app, err := factory.NewApp(cfg)
	if err != nil {
		slog.Error("Failed to initialize application architecture", "error", err)
		os.Exit(1)
	}

	if err := app.Run(); err != nil {
		slog.Error("Application terminated with errors", "error", err)
		os.Exit(1)
	}
}
