package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"net/http"

	"github.com/fedulovivan/device-pinger/internal/counters"
	"github.com/fedulovivan/device-pinger/internal/logger"
	_ "github.com/fedulovivan/device-pinger/internal/logger"
	"github.com/fedulovivan/device-pinger/internal/mqtt"
	"github.com/fedulovivan/device-pinger/internal/registry"
	"github.com/fedulovivan/device-pinger/internal/workers"
	"github.com/fedulovivan/mhz19-go/pkg/utils"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var tag = utils.NewTag(logger.TAG_MAIN)

func main() {

	// record application start time
	registry.RecordStartTime()

	// notify we are in development
	if registry.Config.IsDev {
		slog.Info(tag.F("Running in developlment mode"))
	}

	// connect to mqtt broker
	mqttDisconnect := mqtt.Connect()

	// spawn workers
	for _, target := range registry.Config.TargetIps {
		go func(t string) {
			_, err := workers.Create(t, mqtt.SendStatus)
			if err != nil {
				counters.Errors.Inc()
				slog.Error(tag.F("Unable to create worker"), "err", err.Error())
			}
		}(target)
	}

	// dedicated prometheus http endpoint,
	// since this app has no other http apis
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		_ = http.ListenAndServe(fmt.Sprintf(":%d", registry.Config.PrometheusPort), nil)
	}()

	// handle shutdown
	stopped := make(chan os.Signal, 1)
	signal.Notify(stopped, os.Interrupt, syscall.SIGTERM)
	<-stopped
	slog.Debug(tag.F("App termination signal received"))
	workers.StopAll()

	// wait for the all workers to complete
	slog.Debug(tag.F("Waiting for the all workers to complete"))
	workers.Wait()

	// disconnect mqtt only after stopping workers
	mqttDisconnect()

	slog.Info(tag.F("All done, bye-bye"))

}
