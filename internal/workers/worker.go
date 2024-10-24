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

type TargetAddr string

type OnlineStatusChangeHandler func(
	target TargetAddr,
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
	target          TargetAddr
	pinger          *probing.Pinger
	status          OnlineStatus
	lastSeen        time.Time
	onlineChecker   *time.Ticker
	periodicUpdater *time.Ticker
	done            chan struct{}
	invalid         bool
	tag             utils.Tag
}

func (worker *Worker) Tag() utils.Tag {
	return worker.tag
}

func (worker *Worker) Done() <-chan struct{} {
	return worker.done
}

func (worker *Worker) Stop() {
	worker.Lock()
	defer worker.Unlock()
	slog.Debug(worker.tag.F("Stopping..."))
	worker.pinger.Stop()
	worker.onlineChecker.Stop()
	worker.periodicUpdater.Stop()
	worker.update_status_unsafe(STATUS_UNKNOWN, UPD_SOURCE_WORKER_STOP)
	slog.Info(worker.tag.F("Stopped"))
	close(worker.done)
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
	target TargetAddr,
	onStatusChange OnlineStatusChangeHandler,
) (*Worker, error) {

	// create instance
	worker := &Worker{
		target:         target,
		status:         STATUS_UNKNOWN,
		onStatusChange: onStatusChange,
		tag:            tagBase.With("Ip=%s", target),
		done:           make(chan struct{}),
	}

	var err error
	worker.pinger, err = probing.NewPinger(string(target))
	if err != nil {
		counters.Errors.Inc()
		slog.Error(worker.tag.F("Failed to complete probing.NewPinger()"), "err", err)
		worker.invalid = true
	}
	worker.pinger.Interval = registry.Config.PingerInterval

	// use logger adapter to write pinger messages with slog and with custom prefix
	worker.pinger.SetLogger(SlogAdapter{worker.tag})

	// start periodic checks to ensure device is still online
	worker.onlineChecker = time.NewTicker(
		registry.Config.OfflineCheckInterval,
	)
	go func() {
		for {
			select {
			case <-worker.done:
				return
			case <-worker.onlineChecker.C:
				worker.Lock()
				fmt.Println("onlineChecker tick")
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
		}
	}()

	// start periodic updater
	worker.periodicUpdater = time.NewTicker(
		registry.Config.PeriodicUpdateInterval,
	)
	go func() {
		for {
			select {
			case <-worker.done:
				return
			case <-worker.periodicUpdater.C:
				worker.Lock()
				fmt.Println("periodicUpdater tick")
				worker.onStatusChange(worker.target, worker.status, worker.lastSeen, UPD_SOURCE_PERIODIC /* , worker.tag */)
				worker.Unlock()
			}
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
			err = worker.pinger.Run()
			if err != nil {
				counters.Errors.Inc()
				slog.Error(worker.tag.F("Failed to complete pinger.Run()"), "err", err)
				worker.invalid = true
			}
		}
	}()

	slog.Info(worker.tag.F("Created"))

	return worker, nil
}
