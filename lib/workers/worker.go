package workers

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/fedulovivan/device-pinger/lib/config"
	probing "github.com/prometheus-community/pro-bing"
)

const (
	STATUS_UNKNOWN OnlineStatus = -1
	STATUS_OFFLINE OnlineStatus = 0
	STATUS_ONLINE  OnlineStatus = 1
)

var STATUS_NAMES = map[OnlineStatus]string{
	STATUS_UNKNOWN: "UNKNOWN",
	STATUS_OFFLINE: "OFFLINE",
	STATUS_ONLINE:  "ONLINE",
}

type OnlineStatus int8
type OnlineStatusChangeHandler func(target string, status OnlineStatus)

type Worker struct {
	target          string
	pinger          *probing.Pinger
	status          OnlineStatus
	lastSeen        time.Time
	onlineChecker   *time.Ticker
	periodicUpdater *time.Ticker
	lock            sync.Mutex
	stopped         bool
	invalid         bool
}

func (worker *Worker) Stop(onChange OnlineStatusChangeHandler) {
	worker.lock.Lock()
	log.Printf("[WORKER:%v] Calling Stop()\n", worker.target)
	if worker.stopped {
		log.Fatalf("[WORKER:%v] Already stopped!", worker.target)
		return
	}
	worker.pinger.Stop()
	worker.onlineChecker.Stop()
	worker.periodicUpdater.Stop()
	worker.stopped = true
	worker.UpdateStatus(STATUS_UNKNOWN, "Worker.Stop", onChange)
	worker.lock.Unlock()
	Wg.Done()
}

func (worker *Worker) Status() OnlineStatus {
	return worker.status
}

// (!) update is not protected by lock, since it is expected to be exetrnal
func (worker *Worker) UpdateStatus(status OnlineStatus, updSource string, onChange OnlineStatusChangeHandler) {
	if status != worker.status {
		log.Printf(
			"[WORKER:%v] updSource=%v status=%v\n",
			worker.target,
			updSource,
			STATUS_NAMES[status],
		)
		onChange(worker.target, status)
		worker.status = status
	}
}

func Create(target string, onChange OnlineStatusChangeHandler) *Worker {

	cfg := config.GetInstance()

	Wg.Add(1)

	if Errors == nil {
		Errors = make(chan error)
	}

	// create instance
	worker := Worker{
		target: target,
		status: STATUS_UNKNOWN,
	}

	// provide custom logger to Pinger, to write messages with "[WORKER:<ip>]..." prefix
	mylogger := WorkerLogger{
		Logger: log.New(log.Writer(), log.Prefix(), log.Flags()),
		target: target,
	}

	var err error
	worker.pinger, err = probing.NewPinger(target)
	if err != nil {
		Errors <- fmt.Errorf("[WORKER:%v] failed to complete probing.NewPinger() %v", target, err)
		worker.invalid = true
	}
	worker.pinger.Interval = cfg.PingerInterval
	worker.pinger.SetLogger(mylogger)

	// start periodic checks to ensure device is still online
	worker.onlineChecker = time.NewTicker(
		cfg.OfflineCheckInterval,
	)
	go func() {
		for range worker.onlineChecker.C {
			worker.lock.Lock()
			status := STATUS_OFFLINE
			if time.Now().Before(worker.lastSeen.Add(cfg.OfflineAfter)) {
				status = STATUS_ONLINE
			}
			worker.UpdateStatus(status, "Worker.OnlineChecker", onChange)
			worker.lock.Unlock()
		}
	}()

	// start periodic updater
	worker.periodicUpdater = time.NewTicker(
		cfg.PeriodicUpdateInterval,
	)
	go func() {
		for range worker.periodicUpdater.C {
			onChange(worker.target, worker.Status())
		}
	}()

	// update status and lastSeen
	worker.pinger.OnRecv = func(pkt *probing.Packet) {
		worker.lock.Lock()
		worker.UpdateStatus(STATUS_ONLINE, "Pinger.OnRecv", onChange)
		worker.lastSeen = time.Now()
		worker.lock.Unlock()
	}

	go func() {
		if worker.invalid {
			Errors <- fmt.Errorf("[WORKER:%v] Cannot run pinger since worker already marked as invalid", target)
		} else {
			err = worker.pinger.Run()
			if err != nil {
				Errors <- fmt.Errorf("[WORKER:%v] Failed to complete pinger.Run(): %v", target, err)
				worker.invalid = true
			}
		}
	}()

	log.Printf("[WORKER:%v] Created\n", worker.target)

	return &worker
}
