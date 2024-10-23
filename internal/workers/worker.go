package workers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/fedulovivan/device-pinger/internal/counters"
	"github.com/fedulovivan/device-pinger/internal/logger"
	"github.com/fedulovivan/device-pinger/internal/registry"
	"github.com/fedulovivan/mhz19-go/pkg/utils"
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

var tagBase = utils.NewTag(logger.TAG_WRKR)

type OnlineStatus int8

type OnlineStatusChangeHandler func(
	target string,
	status OnlineStatus,
	lastSeen time.Time,
	updSource UpdSource,
)

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
	sync.Mutex
	onStatusChange  OnlineStatusChangeHandler
	target          string
	pinger          *probing.Pinger
	status          OnlineStatus
	lastSeen        time.Time
	onlineChecker   *time.Ticker
	periodicUpdater *time.Ticker
	stopped         bool
	invalid         bool
	tag             utils.Tag
}

func (worker *Worker) Tag() utils.Tag {
	return worker.tag
}

func (worker *Worker) Stop() {
	worker.Lock()
	defer worker.Unlock()
	slog.Debug(worker.tag.F("Stopping..."))
	if worker.stopped {
		slog.Warn(worker.tag.F("Already stopped!"))
		return
	}
	worker.pinger.Stop()
	worker.onlineChecker.Stop()
	worker.periodicUpdater.Stop()
	worker.stopped = true
	worker.update_status_unsafe(STATUS_UNKNOWN, UPD_SOURCE_WORKER_STOP)
	slog.Info(worker.tag.F("Stopped"))
	collectionWg.Done()
}

func (worker *Worker) Status() OnlineStatus {
	return worker.status
}

func (worker *Worker) LastSeen() time.Time {
	return worker.lastSeen
}

func (worker *Worker) update_status_unsafe(status OnlineStatus, updSource UpdSource) {
	if status != worker.status {
		slog.Debug(
			worker.tag.F("Status changed"),
			"source",
			updSource,
			"status",
			STATUS_NAMES[status],
		)
		worker.onStatusChange(worker.target, status, worker.lastSeen, updSource /* , worker.tag */)
		worker.status = status
	}
}

func New(
	target string,
	onStatusChange OnlineStatusChangeHandler,
) (*Worker, error) {

	tag := tagBase.With("Ip=%s", target)

	// create instance
	worker := &Worker{
		target:         target,
		status:         STATUS_UNKNOWN,
		onStatusChange: onStatusChange,
		tag:            tag,
	}

	// provide custom logger to Pinger, to write messages with "[WORKER:<ip>]..." prefix
	mylogger := WorkerLogger{
		Logger: slog.Default(),
		tag:    tag,
	}

	var err error
	worker.pinger, err = probing.NewPinger(target)
	if err != nil {
		counters.Errors.Inc()
		slog.Error(worker.tag.F("Failed to complete probing.NewPinger()"), "err", err)
		worker.invalid = true
		// worker.status = STATUS_INVALID
	}
	worker.pinger.Interval = registry.Config.PingerInterval
	worker.pinger.SetLogger(mylogger)
	// worker.pinger.SetAddr("1.1.1.1")

	// start periodic checks to ensure device is still online
	worker.onlineChecker = time.NewTicker(
		registry.Config.OfflineCheckInterval,
	)
	go func() {
		for range worker.onlineChecker.C {
			worker.Lock()
			status := STATUS_UNKNOWN
			if !worker.lastSeen.IsZero() {
				if time.Now().Before(worker.lastSeen.Add(registry.Config.OfflineAfter)) {
					status = STATUS_ONLINE
				} else {
					status = STATUS_OFFLINE
				}
			}
			worker.update_status_unsafe(status, UPD_SOURCE_ONLINE_CHECKER)
			worker.Unlock()
		}
	}()

	// start periodic updater
	worker.periodicUpdater = time.NewTicker(
		registry.Config.PeriodicUpdateInterval,
	)
	go func() {
		for range worker.periodicUpdater.C {
			worker.Lock()
			worker.onStatusChange(worker.target, worker.status, worker.lastSeen, UPD_SOURCE_PERIODIC /* , worker.tag */)
			worker.Unlock()
		}
	}()

	// update status and lastSeen
	worker.pinger.OnRecv = func(pkt *probing.Packet) {
		worker.Lock()
		defer worker.Unlock()
		worker.lastSeen = time.Now()
		worker.update_status_unsafe(STATUS_ONLINE, UPD_SOURCE_PING_ON_RECV)
	}

	go func() {
		if worker.invalid {
			counters.Errors.Inc()
			slog.Error(worker.tag.F("Cannot run pinger since worker already marked as invalid"))
		} else {
			err = worker.pinger.Run() /* RunWithContext */
			if err != nil {
				counters.Errors.Inc()
				slog.Error(worker.tag.F("Failed to complete pinger.Run()"), "err", err)
				worker.invalid = true
				// worker.status = STATUS_INVALID
			}
		}
	}()

	slog.Info(worker.tag.F("Created"))

	return worker, nil
}
