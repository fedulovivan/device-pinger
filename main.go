package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"net/http"

	"github.com/fedulovivan/device-pinger/internal/counters"
	_ "github.com/fedulovivan/device-pinger/internal/logger"
	"github.com/fedulovivan/device-pinger/internal/mqtt"
	"github.com/fedulovivan/device-pinger/internal/registry"
	"github.com/fedulovivan/device-pinger/internal/utils"
	"github.com/fedulovivan/device-pinger/internal/workers"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {

	// record application start time
	registry.RecordStartTime()

	// notify we are in development
	if registry.Config.IsDev {
		slog.Info("[MAIN] running in developlment mode")
	}

	// print mem usage on startup
	utils.PrintMemUsage()

	// connect to mqtt broker
	mqttDisconnect := mqtt.Connect()

	// send first update with initial stats
	mqtt.SendStats()

	// spawn workers
	for _, target := range registry.Config.TargetIps {
		go func(t string) {
			_, err := workers.Create(t, mqtt.SendStatus)
			if err != nil {
				counters.Errors.Inc()
				slog.Error("[MAIN] unable to create worker", "err", err.Error())
			}
		}(target)
	}

	// prometheus
	http.Handle("/metrics", promhttp.Handler())
	_ = http.ListenAndServe(fmt.Sprintf(":%d", registry.Config.PrometheusPort), nil)

	// handle shutdown
	appStopped := make(chan bool)
	signalsCh := make(chan os.Signal, 1)
	signal.Notify(signalsCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		for range signalsCh {
			appStopped <- true
		}
	}()
	<-appStopped
	slog.Debug("[MAIN] app termination signal received")
	workers.StopAll()

	// wait for the all workers to complete
	slog.Debug("[MAIN] waiting for the all workers to complete")
	workers.Wait()

	// disconnect mqtt only after stopping workers
	mqttDisconnect()

	slog.Info("[MAIN] all done, bye-bye")

}
