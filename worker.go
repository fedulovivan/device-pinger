package main

import (
	"device-pinger/constants"
	"fmt"
	"log"
	"sync"
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

var workers []*Worker

// var workersLock sync.Mutex
var workersWg sync.WaitGroup

type Worker struct {
	target        string
	pinger        *probing.Pinger
	online        bool
	lastSeen      time.Time
	onlineChecker *time.Ticker
	lock          sync.Mutex
	stopped       bool
}

type OnlineStatusChangeHandler func(target string, online bool)

func (worker *Worker) Stop() {
	worker.lock.Lock()
	log.Printf("[WRKR:%v] calling Stop()\n", worker.target)
	if worker.stopped {
		log.Fatalf("[WRKR:%v] already stopped!", worker.target)
		return
	}
	worker.pinger.Stop()
	worker.onlineChecker.Stop()
	worker.stopped = true
	workersWg.Done()
	worker.lock.Unlock()
}

// (!) update is not protected by lock, which is expected to be exetrnal
func (worker *Worker) UpdateOnline(online bool, updSource string, onChange OnlineStatusChangeHandler) {
	if online != worker.online {

		textStatus := "OFFLINE"
		if online && !worker.online {
			textStatus = "ONLINE"
		}

		log.Printf(
			"[WRKR:%v] updSource=%v status=%v\n",
			worker.target,
			updSource,
			textStatus,
		)

		onChange(worker.target, online)

		worker.online = online
	}
}

func SpawnWorker(target string, errorsCh chan<- error, onChange OnlineStatusChangeHandler) *Worker {

	pinger, err := probing.NewPinger(target)
	if err != nil {
		errorsCh <- fmt.Errorf("[WRKR:%v] failed to complete probing.NewPinger() %v", target, err)
	}
	pinger.Interval = constants.PINGER_INTERVAL

	ticker := time.NewTicker(constants.OFFLINE_CHECK_INTERVAL)

	worker := Worker{
		target:        target,
		pinger:        pinger,
		online:        false,
		lastSeen:      time.Time{},
		onlineChecker: ticker,
		lock:          sync.Mutex{},
		stopped:       false,
	}

	// start periodic checks to ensure device is still online
	go func() {
		for range ticker.C {
			worker.lock.Lock()
			online := time.Now().Before(worker.lastSeen.Add(constants.OFFLINE_AFTER))
			worker.UpdateOnline(online, "Worker.Ticker", onChange)
			worker.lock.Unlock()
		}
	}()

	// update online and lastSeen
	pinger.OnRecv = func(pkt *probing.Packet) {
		worker.lock.Lock()
		worker.UpdateOnline(true, "Pinger.OnRecv", onChange)
		worker.lastSeen = time.Now()
		worker.lock.Unlock()
	}

	go func() {
		err = pinger.Run()
		if err != nil {
			errorsCh <- fmt.Errorf("[WRKR:%v] failed to complete pinger.Run(): %v", target, err)
		}
	}()

	return &worker
}
