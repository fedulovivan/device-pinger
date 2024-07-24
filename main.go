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
		for e := range workers.Errors {
			log.Println(e)
		}
	}()

	// add extra wg item to keep app runnnig with zero workers
	workers.Wg.Add(1)
	// spawn workers
	for _, target := range cfg.TargetIps {
		workers.Add(workers.Create(
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
			for _, worker := range workers.GetAsList() {
				worker.Stop(mqttclient.HandleOnlineChange)
			}
			workers.Wg.Done()
		}
		close(workers.Errors)
	}()

	// infinetly wait for workers to complete
	workers.Wg.Wait()

	log.Println("[MAIN] all done, bye-bye")

}
