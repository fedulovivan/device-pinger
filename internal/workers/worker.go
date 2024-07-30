package workers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/fedulovivan/device-pinger/internal/config"
	probing "github.com/prometheus-community/pro-bing"
)

const (
	STATUS_INVALID OnlineStatus = -2
	STATUS_UNKNOWN OnlineStatus = -1
	STATUS_OFFLINE OnlineStatus = 0
	STATUS_ONLINE  OnlineStatus = 1
)

var STATUS_NAMES = map[OnlineStatus]string{
	STATUS_INVALID: "invalid",
	STATUS_UNKNOWN: "unknown",
	STATUS_OFFLINE: "offline",
	STATUS_ONLINE:  "online",
}

type OnlineStatus int8

type OnlineStatusChangeHandler func(target string, status OnlineStatus, lastSeen time.Time, updSource UpdSource)

type UpdSource byte

func (s UpdSource) String() string {
	return fmt.Sprintf("%v (id=%d)", UPD_SOURCE_NAMES[s], s)
}

func (s UpdSource) MarshalJSON() (b []byte, err error) {
	return json.Marshal(s.String())
}

const (
	UPD_SOURCE_MQTT_GET       UpdSource = 1
	UPD_SOURCE_WORKER_STOP    UpdSource = 2
	UPD_SOURCE_ONLINE_CHECKER UpdSource = 3
	UPD_SOURCE_PERIODIC       UpdSource = 4
	UPD_SOURCE_PING_ON_RECV   UpdSource = 5
)

var UPD_SOURCE_NAMES = map[UpdSource]string{
	UPD_SOURCE_MQTT_GET:       "mqtt get",
	UPD_SOURCE_WORKER_STOP:    "worker stop",
	UPD_SOURCE_ONLINE_CHECKER: "online checker",
	UPD_SOURCE_PERIODIC:       "periodic updater",
	UPD_SOURCE_PING_ON_RECV:   "ping onrecv",
}

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
	defer worker.lock.Unlock()

	slog.Info(worker.LogTag("Calling Stop()"))
	if worker.stopped {
		slog.Warn(worker.LogTag("Already stopped!"))
		return
	}
	worker.pinger.Stop()
	worker.onlineChecker.Stop()
	worker.periodicUpdater.Stop()
	worker.stopped = true
	worker.UpdateStatus(STATUS_UNKNOWN, UPD_SOURCE_WORKER_STOP, onChange)
	Wg.Done()
}

func (worker *Worker) Status() OnlineStatus {
	return worker.status
}

func (worker *Worker) LastSeen() time.Time {
	return worker.lastSeen
}

func (worker *Worker) LogTag(message string) string {
	return fmt.Sprintf("[WORKER:%v] %v", worker.target, message)
}

// (!) update is not protected by lock, which is expected to be external
func (worker *Worker) UpdateStatus(status OnlineStatus, updSource UpdSource, onChange OnlineStatusChangeHandler) {
	// if worker.invalid {
	// 	slog.Error(worker.LogTag("Unexpected call of UpdateStatus() for worker in invalid status"))
	// 	return
	// }
	if status != worker.status {
		slog.Debug(
			worker.LogTag("Status changed"),
			"SOURCE",
			updSource,
			"STATUS",
			STATUS_NAMES[status],
		)
		onChange(worker.target, status, worker.lastSeen, updSource)
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
		Logger: slog.Default(),
		target: target,
	}

	var err error
	worker.pinger, err = probing.NewPinger(target)
	if err != nil {
		Errors <- fmt.Errorf(worker.LogTag("Failed to complete probing.NewPinger()"), err)
		worker.invalid = true
		// worker.status = STATUS_INVALID
	}
	worker.pinger.Interval = cfg.PingerInterval
	worker.pinger.SetLogger(mylogger)
	// worker.pinger.SetAddr("1.1.1.1")

	// start periodic checks to ensure device is still online
	worker.onlineChecker = time.NewTicker(
		cfg.OfflineCheckInterval,
	)
	go func() {
		for range worker.onlineChecker.C {
			worker.lock.Lock()
			status := STATUS_UNKNOWN
			if !worker.lastSeen.IsZero() {
				if time.Now().Before(worker.lastSeen.Add(cfg.OfflineAfter)) {
					status = STATUS_ONLINE
				} else {
					status = STATUS_OFFLINE
				}
			}
			worker.UpdateStatus(status, UPD_SOURCE_ONLINE_CHECKER, onChange)
			worker.lock.Unlock()
		}
	}()

	// start periodic updater
	worker.periodicUpdater = time.NewTicker(
		cfg.PeriodicUpdateInterval,
	)
	go func() {
		for range worker.periodicUpdater.C {
			onChange(worker.target, worker.status, worker.lastSeen, UPD_SOURCE_PERIODIC)
		}
	}()

	// update status and lastSeen
	worker.pinger.OnRecv = func(pkt *probing.Packet) {
		worker.lock.Lock()
		defer worker.lock.Unlock()
		worker.lastSeen = time.Now()
		worker.UpdateStatus(STATUS_ONLINE, UPD_SOURCE_PING_ON_RECV, onChange)
	}

	go func() {
		if worker.invalid {
			Errors <- fmt.Errorf(worker.LogTag("Cannot run pinger since worker already marked as invalid"))
		} else {
			err = worker.pinger.Run()
			if err != nil {
				Errors <- fmt.Errorf(worker.LogTag("Failed to complete pinger.Run()"), err)
				worker.invalid = true
				// worker.status = STATUS_INVALID
			}
		}
	}()

	slog.Info(worker.LogTag("Created"))

	return &worker
}
