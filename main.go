package main

import (
	"log/slog"
	"os"

	"github.com/hantabaru1014/instant-playback-sync/app"
)

func main() {
	loglvl := slog.LevelInfo
	switch os.Getenv("LOG_LEVEL") {
	case "DEBUG":
		loglvl = slog.LevelDebug
	case "INFO":
		loglvl = slog.LevelInfo
	case "WARN":
		loglvl = slog.LevelWarn
	case "ERROR":
		loglvl = slog.LevelError
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: loglvl,
	}))
	slog.SetDefault(logger)

	s := app.NewServer()
	s.Run(":8080")
}
