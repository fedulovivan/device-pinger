package main

import (
	"log"
	"os"
	"os/signal"
	"strings"

	"device-pinger/lib/mqttclient"
	"device-pinger/lib/workers"

	_ "github.com/joho/godotenv/autoload"
)

func main() {

	// mqtt
	mqttclient.Init()

	// immediately pull and print all emitted worker errors
	go func() {
		for e := range workers.GetErrors() {
			log.Println(e)
		}
	}()

	// spawn workers
	tt := strings.Split(os.Getenv("TARGET_IPS"), ",")
	for _, target := range tt {
		workers.Push(workers.Create(
			target,
			mqttclient.HandleOnlineChange,
		))
	}

	// handle program interrupt
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
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
