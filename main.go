package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/fedulovivan/device-pinger/internal/config"
	"github.com/fedulovivan/device-pinger/internal/mqtt"
	"github.com/fedulovivan/device-pinger/internal/utils"
	"github.com/fedulovivan/device-pinger/internal/workers"
)

func main() {

	// dump initial mem usage
	utils.PrintMemUsage()

	// get config struct
	cfg := config.GetInstance()

	// update logger level from config
	slog.SetLogLoggerLevel(cfg.LogLevel)

	// mqtt
	mqtt.Init()

	// immediately pull and print all emitted worker errors
	go func() {
		for e := range workers.Errors {
			slog.Error(e.Error())
		}
	}()

	// add extra wg item to keep app runnnig with zero workers
	workers.Wg.Add(1)
	// spawn workers
	for _, target := range cfg.TargetIps {
		workers.Add(workers.Create(
			target,
			mqtt.SendStatus,
		))
		utils.PrintMemUsage()
	}
	mqtt.SendStats()

	// handle program interrupt
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, os.Interrupt, syscall.SIGTERM)
	go func() {
		for range sc {
			slog.Info(fmt.Sprintf(
				"[MAIN] Interrupt signal captured, stopping %v worker(s)...",
				workers.GetCount(),
			))
			for _, worker := range workers.GetAsList() {
				worker.Stop(mqtt.SendStatus)
			}
			workers.Wg.Done()
		}
		close(workers.Errors)
	}()

	// infinetly wait for workers to complete
	workers.Wg.Wait()

	slog.Info("[MAIN] all done, bye-bye")

}
