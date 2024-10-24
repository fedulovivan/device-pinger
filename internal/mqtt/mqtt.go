package mqtt

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/fedulovivan/device-pinger/internal/counters"
	"github.com/fedulovivan/device-pinger/internal/logger"
	"github.com/fedulovivan/device-pinger/internal/registry"
	"github.com/fedulovivan/device-pinger/internal/workers"
	"github.com/fedulovivan/mhz19-go/pkg/utils"

	MqttLib "github.com/eclipse/paho.mqtt.golang"
)

var LBRACKET = byte('{')

var tagBase = utils.NewTag(logger.TAG_MQTT)

type Request struct {
	Seq int `json:"seq"`
}

type SequencedResponse struct {
	Seq     int    `json:"seq"`
	Message string `json:"message"`
	IsError bool   `json:"error"`
}

type StatusResponse struct {
	Status    workers.OnlineStatus `json:"status"`
	LastSeen  time.Time            `json:"lastSeen"`
	UpdSource workers.UpdSource    `json:"updSource"`
}

type StatsResponse struct {
	Workers     int             `json:"workers"`
	MemoryAlloc uint64          `json:"memoryAlloc"`
	Uptime      registry.Uptime `json:"uptime"`
}

var client MqttLib.Client

var workersCollection *workers.Collection

func GetBokerString() string {
	return fmt.Sprintf("tcp://%s:%d", registry.Config.MqttHost, registry.Config.MqttPort)
}

func Connect(wc *workers.Collection) func() {
	workersCollection = wc
	opts := MqttLib.NewClientOptions()
	opts.AddBroker(GetBokerString())
	opts.SetClientID(registry.Config.MqttClientId)
	opts.SetUsername(registry.Config.MqttUsername)
	opts.SetPassword(registry.Config.MqttPassword)
	opts.SetDefaultPublishHandler(defaultMessageHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	client = MqttLib.NewClient(opts)
	slog.Debug(tagBase.F("Connecting..."))
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		counters.Errors.Inc()
		slog.Error(tagBase.F("Error"), "error", token.Error())
	} else {
		subscribeAll(client)
	}
	return func() {
		slog.Debug(tagBase.F("Disconnect..."))
		client.Disconnect(250)
	}
}

func BuildTopic(target workers.TargetAddr, action string) string {
	parts := []string{registry.Config.MqttTopicBase}
	if len(target) == 0 {
		parts = append(parts, action)
	} else {
		parts = append(parts, string(target), action)
	}
	return strings.Join(parts, "/")
}

func Publish(target workers.TargetAddr, action string, rsp any /* , tagext utils.Tag */) error {
	payload, err := json.Marshal(rsp)
	if err != nil {
		return err
	}
	topic := BuildTopic(target, action)
	token := client.Publish(
		topic,
		0,
		false,
		payload,
	)
	token.Wait()
	if token.Error() != nil {
		return token.Error()
	}
	slog.Debug(tagBase.F("Published"), "topic", topic, "payload", string(payload))
	counters.MqttPublished.Inc()
	return nil
}

func SendStats() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	rsp := StatsResponse{
		MemoryAlloc: m.Alloc,
		Workers:     workersCollection.Len(),
		Uptime:      registry.GetUptime(),
	}
	err := Publish("", "stats", rsp)
	if err != nil {
		counters.Errors.Inc()
		slog.Error(tagBase.F("Error"), "err", err)
	}
}

var SendStatus workers.OnlineStatusChangeHandler = func(
	target workers.TargetAddr,
	status workers.OnlineStatus,
	lastSeen time.Time,
	updSource workers.UpdSource,
) {
	rsp := StatusResponse{
		Status:    status,
		LastSeen:  lastSeen,
		UpdSource: updSource,
	}
	err := Publish(target, "status", rsp)
	if err != nil {
		counters.Errors.Inc()
		slog.Error(tagBase.F("Error"), "err", err)
	}
}

func SendOpFeedback(req *Request, target workers.TargetAddr, message string, isError bool) {
	rsp := SequencedResponse{
		Message: message,
		IsError: isError,
		Seq:     req.Seq,
	}
	if isError {
		counters.Errors.Inc()
		slog.Error(tagBase.F("Error"), "err", message)
	}
	err := Publish(target, "rsp", rsp)
	if err != nil {
		counters.Errors.Inc()
		slog.Error(tagBase.F("Error"), "err", err)
	}
}

func dispatchAction(action string, target workers.TargetAddr, req *Request) {
	handled := false
	switch action {
	case "get-stats":
		SendStats()
		handled = true
	case "get":
		slog.Debug(tagBase.F("Getting status for %v", target))
		worker, err := workersCollection.Get(target)
		if err == nil {
			SendStatus(target, worker.Status(), worker.LastSeen(), workers.UPD_SOURCE_MQTT_GET)
			handled = true
		} else {
			SendOpFeedback(req, target, err.Error(), true)
		}
	case "add":
		slog.Debug(tagBase.F("Adding new worker for %v", target))
		_, err := workersCollection.Create(
			target,
			SendStatus,
		)
		if err == nil {
			SendOpFeedback(req, target, "added", false)
			SendStats()
			handled = true
		} else {
			SendOpFeedback(req, target, err.Error(), true)
		}
	case "del":
		slog.Debug(tagBase.F("Deleting worker for %v", target))
		err := workersCollection.Delete(target, SendStatus)
		runtime.GC()
		if err == nil {
			SendOpFeedback(req, target, "deleted", false)
			SendStats()
			handled = true
		} else {
			SendOpFeedback(req, target, err.Error(), true)
		}
	}
	if handled {
		counters.ActionsHandled.WithLabelValues(action, string(target)).Inc()
	}
}

var defaultMessageHandler MqttLib.MessageHandler = func(client MqttLib.Client, msg MqttLib.Message) {

	counters.MqttReceived.Inc()

	topic := msg.Topic()

	slog.Debug(
		tagBase.F("Received"),
		"topic", topic,
		"payload", utils.Truncate(string(msg.Payload()), 80),
	)

	tt := strings.Split(topic, "/")

	ttlen := len(tt)

	message := Request{}

	tryAsJson := len(msg.Payload()) > 0 && msg.Payload()[0] == LBRACKET

	if tryAsJson {
		err := json.Unmarshal(msg.Payload(), &message)
		if err != nil {
			counters.Errors.Inc()
			slog.Error(tagBase.F("Failed to parse message payload into json"), "err", err)
		}
	}

	if ttlen == 2 {
		action := tt[1]
		dispatchAction(action, "", &message)
	} else if ttlen == 3 {
		target := workers.TargetAddr(tt[1])
		action := tt[2]
		dispatchAction(action, target, &message)
	} else {
		counters.Errors.Inc()
		slog.Error(tagBase.F("Unexpected topic format %v", topic))
		return
	}

}

var connectHandler MqttLib.OnConnectHandler = func(client MqttLib.Client) {
	slog.Info(tagBase.F("Connected"), "broker", GetBokerString())
}

var connectLostHandler MqttLib.ConnectionLostHandler = func(client MqttLib.Client, err error) {
	counters.Errors.Inc()
	slog.Error(tagBase.F("Connection lost"), "err", err)
}

func subscribeOne(client MqttLib.Client, topic string, wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}
	if token := client.Subscribe(topic, 0, nil); token.Wait() && token.Error() != nil {
		counters.Errors.Inc()
		slog.Error(tagBase.F("client.Subscribe()"), "err", token.Error())
	}
	slog.Info(tagBase.F("Subscribed to"), "topic", topic)
}

func subscribeAll(client MqttLib.Client) {
	suffixes := []string{
		"get-stats",
		"+/add",
		"+/get",
		"+/del",
	}
	var wg sync.WaitGroup
	wg.Add(len(suffixes))
	for _, suffix := range suffixes {
		go subscribeOne(
			client,
			registry.Config.MqttTopicBase+"/"+suffix,
			&wg,
		)
	}
	wg.Wait()
	slog.Debug(tagBase.F("All subscribtions are settled"))
}
