package main

import (
	"log"
	"os"
	"os/signal"

	"github.com/fedulovivan/device-pinger/lib/config"
	"github.com/fedulovivan/device-pinger/lib/mqttclient"
	"github.com/fedulovivan/device-pinger/lib/workers"
)

func main() {

	// get config struct
	cfg := config.GetInstance()

	// mqtt
	mqttclient.Init()

	// immediately pull and print all emitted worker errors
	go func() {
		for e := range workers.GetErrors() {
			log.Println(e)
		}
	}()

	// spawn workers
	for _, target := range cfg.TargetIps {
		workers.Push(workers.Create(
			target,
			mqttclient.HandleOnlineChange,
		))
	}

	// handle program interrupt
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, os.Interrupt)
	go func() {
		for range sc {
			log.Printf(
				"[MAIN] Interrupt signal captured, stopping %v workers...\n",
				workers.GetCount(),
			)
			for _, worker := range workers.Get() {
				worker.Stop()
			}
		}
		close(workers.GetErrors())
	}()

	// infinetly wait for workers to complete
	workers.Wait()

	log.Println("[MAIN] all done, bye-bye")

}
