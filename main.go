package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"device-pinger/constants"
	"device-pinger/mqtt"
)

var handleOnlineChange OnlineStatusChangeHandler = func(target string, online bool) {
	token := mqtt.Client().Publish(
		fmt.Sprintf("device-pinger/%v/status", target),
		0,
		false,
		fmt.Sprintf(`{"online":%v}`, online),
	)
	token.Wait()
}

func main() {

	// mqtt
	mqtt.Init()

	// init chan for async errors
	errorsCh := make(chan error)

	// immediately pull and print errors from chan
	go func() {
		for e := range errorsCh {
			log.Println(e)
		}
	}()

	// spawn workers
	for _, target := range constants.TARGET_IPS {
		workersWg.Add(1)
		worker := SpawnWorker(
			target,
			errorsCh,
			handleOnlineChange,
		)
		workers = append(workers, worker)
	}

	// handle program interrupt
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			log.Printf(
				"[MAIN] Interrupt signal captured, stopping %v workers...\n",
				len(workers),
			)
			for _, worker := range workers {
				worker.Stop()
			}
		}
		close(errorsCh)
	}()

	// infinetly wait for workers to complete
	workersWg.Wait()

	log.Println("[MAIN] all done, bye-bye")

}
