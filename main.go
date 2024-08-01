package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/fedulovivan/device-pinger/internal/logger"
	"github.com/fedulovivan/device-pinger/internal/mqtt"
	"github.com/fedulovivan/device-pinger/internal/registry"
	"github.com/fedulovivan/device-pinger/internal/utils"
	"github.com/fedulovivan/device-pinger/internal/workers"
)

func main() {

	// record application start time
	registry.RecordStartTime()

	// notify we are in development
	if registry.Config.IsDev {
		slog.Warn("[MAIN] running in developlment mode")
	}

	// print mem usage on startup
	utils.PrintMemUsage()

	// connect to mqtt broker
	mqttDisconnect := mqtt.Connect()

	// spawn workers
	for _, target := range registry.Config.TargetIps {
		go func(t string) {
			worker, err := workers.Create(
				t,
				mqtt.SendStatus,
			)
			if err == nil {
				workers.Add(worker)
			} else {
				slog.Error("[MAIN] unable to create worker", "err", err.Error())
			}
		}(target)
	}

	// send first update with initial stats
	mqtt.SendStats()

	// handle shutdown
	signals := make(chan os.Signal, 1)
	stopped := make(chan bool)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		for range signals {
			stopped <- true
		}
	}()
	<-stopped
	slog.Info("[MAIN] app termination signal received")
	workers.StopAll()

	// wait for the all workers to complete
	workers.Wait()

	// disconnect mqtt only after stopping workers
	mqttDisconnect()

	slog.Info("[MAIN] all done, bye-bye")

}
