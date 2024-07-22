package workers

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/fedulovivan/device-pinger/lib/utils"

	probing "github.com/prometheus-community/pro-bing"
)

var workers map[string](*Worker)
var lock sync.RWMutex
var wg sync.WaitGroup
var errorsCh chan error

const (
	OFFLINE_AFTER_DEF          = 30
	PINGER_INTERVAL_DEF        = 5
	OFFLINE_CHECK_INTERVAL_DEF = 5
)

type Worker struct {
	target        string
	pinger        *probing.Pinger
	online        bool
	lastSeen      time.Time
	onlineChecker *time.Ticker
	lock          sync.Mutex
	stopped       bool
	invalid       bool
}

type OnlineStatusChangeHandler func(target string, online bool)

func GetErrors() chan error {
	return errorsCh
}

func Wait() {
	wg.Wait()
}

func Has(target string) bool {
	lock.Lock()
	defer lock.Unlock()
	_, ok := workers[target]
	return ok
}

func Delete(target string) {
	lock.Lock()
	defer lock.Unlock()
	w, ok := workers[target]
	if ok {
		w.Stop()
		delete(workers, target)
		log.Printf("[MAIN] Worker deleted, new size %v", GetCount())
	}
}

func Get() []*Worker {
	return utils.Values(workers)
}

func GetCount() int {
	return len(workers)
}

func Push(worker *Worker) {
	lock.Lock()
	defer lock.Unlock()
	if workers == nil {
		workers = make(map[string]*Worker)
	}
	workers[worker.target] = worker
	log.Printf("[MAIN] Worker pushed, new size %v", GetCount())
}

func (worker *Worker) Stop() {
	worker.lock.Lock()
	log.Printf("[WORKER:%v] Calling Stop()\n", worker.target)
	if worker.stopped {
		log.Fatalf("[WORKER:%v] Already stopped!", worker.target)
		return
	}
	worker.pinger.Stop()
	worker.onlineChecker.Stop()
	worker.stopped = true
	worker.lock.Unlock()
	wg.Done()
}

// (!) update is not protected by lock, since it is expected to be exetrnal
func (worker *Worker) UpdateOnline(online bool, updSource string, onChange OnlineStatusChangeHandler) {
	if online != worker.online {
		textStatus := "OFFLINE"
		if online && !worker.online {
			textStatus = "ONLINE"
		}
		log.Printf(
			"[WORKER:%v] updSource=%v status=%v\n",
			worker.target,
			updSource,
			textStatus,
		)
		onChange(worker.target, online)
		worker.online = online
	}
}

func Create(target string, onChange OnlineStatusChangeHandler) *Worker {

	wg.Add(1)

	if errorsCh == nil {
		errorsCh = make(chan error)
	}

	worker := Worker{
		target:        target,
		pinger:        &probing.Pinger{},
		online:        false,
		lastSeen:      time.Time{},
		onlineChecker: &time.Ticker{},
		lock:          sync.Mutex{},
		stopped:       false,
		invalid:       false,
	}

	// provide custom logger to Pinger, to write messages with "[WORKER:<ip>]..." prefix
	mylogger := WorkerLogger{
		Logger: log.New(log.Writer(), log.Prefix(), log.Flags()),
		target: target,
	}

	var err error
	worker.pinger, err = probing.NewPinger(target)
	if err != nil {
		errorsCh <- fmt.Errorf("[WORKER:%v] failed to complete probing.NewPinger() %v", target, err)
		worker.invalid = true
	}
	worker.pinger.Interval = utils.GetNumericEnv("PINGER_INTERVAL", PINGER_INTERVAL_DEF)
	worker.pinger.SetLogger(mylogger)

	// start periodic checks to ensure device is still online
	worker.onlineChecker = time.NewTicker(utils.GetNumericEnv("OFFLINE_CHECK_INTERVAL", OFFLINE_CHECK_INTERVAL_DEF))
	go func() {
		for range worker.onlineChecker.C {
			worker.lock.Lock()
			online := time.Now().Before(worker.lastSeen.Add(utils.GetNumericEnv("OFFLINE_AFTER", OFFLINE_AFTER_DEF)))
			worker.UpdateOnline(online, "Worker.Ticker", onChange)
			worker.lock.Unlock()
		}
	}()

	// update online and lastSeen
	worker.pinger.OnRecv = func(pkt *probing.Packet) {
		worker.lock.Lock()
		worker.UpdateOnline(true, "Pinger.OnRecv", onChange)
		worker.lastSeen = time.Now()
		worker.lock.Unlock()
	}

	go func() {
		if worker.invalid {
			errorsCh <- fmt.Errorf("[WORKER:%v] Cannot run pinger since worker already marked as invalid", target)
		} else {
			err = worker.pinger.Run()
			if err != nil {
				errorsCh <- fmt.Errorf("[WORKER:%v] Failed to complete pinger.Run(): %v", target, err)
				worker.invalid = true
			}
		}
	}()

	log.Printf("[WORKER:%v] Created\n", worker.target)

	return &worker
}
