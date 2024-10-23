package logger

import (
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/fedulovivan/device-pinger/internal/registry"
	"github.com/fedulovivan/mhz19-go/pkg/utils"
	"github.com/lmittmann/tint"
)

const (
	TAG_MAIN utils.TagName = "[main   ]"
	TAG_MQTT utils.TagName = "[mqtt   ]"
	TAG_WRKR utils.TagName = "[worker ]"
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
		log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
	}
}
