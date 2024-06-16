package main

import (
	"log/slog"
	"os"

	"github.com/hantabaru1014/instant-playback-sync/app"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	s := app.NewServer()
	s.Run(":8080")
}
