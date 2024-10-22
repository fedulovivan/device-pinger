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
	"github.com/fedulovivan/device-pinger/internal/registry"
	"github.com/fedulovivan/device-pinger/internal/utils"
	"github.com/fedulovivan/device-pinger/internal/workers"

	MqttLib "github.com/eclipse/paho.mqtt.golang"
)

type InMessage struct {
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

func GetBokerString() string {
	return fmt.Sprintf("tcp://%s:%d", registry.Config.MqttHost, registry.Config.MqttPort)
}

func Connect() func() {
	opts := MqttLib.NewClientOptions()
	opts.AddBroker(GetBokerString())
	opts.SetClientID(registry.Config.MqttClientId)
	opts.SetUsername(registry.Config.MqttUsername)
	opts.SetPassword(registry.Config.MqttPassword)
	opts.SetDefaultPublishHandler(defaultMessageHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	client = MqttLib.NewClient(opts)
	slog.Debug("[MQTT] Connecting...")
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		counters.Errors.Inc()
		slog.Error("[MQTT]", "error", token.Error())
	} else {
		subscribeAll(client)
	}
	return func() {
		slog.Debug("[MQTT] Disconnect...")
		client.Disconnect(250)
	}
}

func BuildTopic(target string, action string) string {
	parts := []string{registry.Config.MqttTopicBase}
	if len(target) == 0 {
		parts = append(parts, action)
	} else {
		parts = append(parts, target, action)
	}
	return strings.Join(parts, "/")
}

func Publish(target string, action string, rsp any) error {
	resString, err := json.Marshal(rsp)
	if err != nil {
		return err
	}
	token := client.Publish(
		BuildTopic(target, action),
		0,
		false,
		resString,
	)
	token.Wait()
	if token.Error() != nil {
		return token.Error()
	}
	return nil
}

func SendStats() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	rsp := StatsResponse{
		MemoryAlloc: m.Alloc,
		Workers:     workers.Len(),
		Uptime:      registry.GetUptime(),
	}
	err := Publish("", "stats", rsp)
	if err != nil {
		counters.Errors.Inc()
		slog.Error("[MQTT]", "err", err)
	}
}

var SendStatus workers.OnlineStatusChangeHandler = func(
	target string,
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
		slog.Error("[MQTT]", "err", err)
	}
}

func SendOpFeedback(req *InMessage, target string, message string, isError bool) {
	rsp := SequencedResponse{
		Message: message,
		IsError: isError,
		Seq:     req.Seq,
	}
	err := Publish(target, "rsp", rsp)
	if err != nil {
		counters.Errors.Inc()
		slog.Error("[MQTT]", "err", err)
	}
}

func hadleAction(action string, target string, message *InMessage) {
	switch action {
	case "get-stats":
		SendStats()
	case "get":
		slog.Debug("[MQTT] Getting status for " + target)
		worker, err := workers.Get(target)
		if err == nil {
			SendStatus(target, worker.Status(), worker.LastSeen(), workers.UPD_SOURCE_MQTT_GET)
		} else {
			SendOpFeedback(message, target, err.Error(), true)
		}
	case "add":
		slog.Debug("[MQTT] Adding new worker for " + target)
		_, err := workers.Create(
			target,
			SendStatus,
		)
		if err == nil {
			SendOpFeedback(message, target, "added", false)
			SendStats()
		} else {
			SendOpFeedback(message, target, err.Error(), true)
		}
	case "del":
		slog.Debug("[MQTT] Deleting worker for " + target)
		err := workers.Delete(target, SendStatus)
		if err == nil {
			SendOpFeedback(message, target, "deleted", false)
			SendStats()
		} else {
			SendOpFeedback(message, target, err.Error(), true)
		}
	}
}

var defaultMessageHandler MqttLib.MessageHandler = func(client MqttLib.Client, msg MqttLib.Message) {

	topic := msg.Topic()

	slog.Debug(
		"[MQTT]",
		"TOPIC", topic,
		"MESSAGE", utils.Truncate(string(msg.Payload()), 80),
	)

	tt := strings.Split(topic, "/")

	ttlen := len(tt)

	message := InMessage{}

	if len(msg.Payload()) > 0 && msg.Payload()[0] == 123 /* 123 = "{" */ {
		err := json.Unmarshal(msg.Payload(), &message)
		if err != nil {
			counters.Errors.Inc()
			slog.Error("[MQTT]", "err", err)
		}
	}

	counters.ApiRequests.WithLabelValues(topic).Inc()

	if ttlen == 2 {
		action := tt[1]
		hadleAction(action, "", &message)
	} else if ttlen == 3 {
		target := tt[1]
		action := tt[2]
		hadleAction(action, target, &message)
	} else {
		counters.Errors.Inc()
		slog.Error("[MQTT] Unexpected topic format " + topic)
		return
	}

}

var connectHandler MqttLib.OnConnectHandler = func(client MqttLib.Client) {
	slog.Info("[MQTT] Connected", "broker", GetBokerString())
}

var connectLostHandler MqttLib.ConnectionLostHandler = func(client MqttLib.Client, err error) {
	counters.Errors.Inc()
	slog.Error("[MQTT] Connection lost", "err", err)
}

func subscribe(client MqttLib.Client, topic string, wg *sync.WaitGroup) {
	defer wg.Done()
	if token := client.Subscribe(topic, 0, nil); token.Wait() && token.Error() != nil {
		counters.Errors.Inc()
		slog.Error("[MQTT] client.Subscribe()", "err", token.Error())
	}
	slog.Info("[MQTT] Subscribed", "topic", topic)
}

func subscribeAll(client MqttLib.Client) {
	var wg sync.WaitGroup
	for _, topic := range []string{registry.Config.MqttTopicBase + "/#"} {
		wg.Add(1)
		go subscribe(client, topic, &wg)
	}
	wg.Wait()
	slog.Debug("[MQTT] All subscribtions are settled")
}
