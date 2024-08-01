package logger

import (
	"log/slog"
	"os"
	"time"

	"github.com/fedulovivan/device-pinger/internal/registry"
	"github.com/lmittmann/tint"
)

func init() {
	if registry.Config.IsDev {
		w := os.Stderr
		slog.SetDefault(slog.New(
			tint.NewHandler(w, &tint.Options{
				Level:      registry.Config.LogLevel,
				TimeFormat: time.TimeOnly,
			}),
		))
	} else {
		slog.SetLogLoggerLevel(registry.Config.LogLevel)
	}
}
