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
	utils_local "github.com/fedulovivan/device-pinger/internal/utils"
	"github.com/fedulovivan/device-pinger/internal/workers"
	"github.com/fedulovivan/mhz19-go/pkg/utils"

	MqttLib "github.com/eclipse/paho.mqtt.golang"
)

var LBRACKET = byte('{')

var tagBase = utils.NewTag(logger.TAG_MQTT)

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

func BuildTopic(target string, action string) string {
	parts := []string{registry.Config.MqttTopicBase}
	if len(target) == 0 {
		parts = append(parts, action)
	} else {
		parts = append(parts, target, action)
	}
	return strings.Join(parts, "/")
}

func Publish(target string, action string, rsp any /* , tagext utils.Tag */) error {
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
		Workers:     workers.Len(),
		Uptime:      registry.GetUptime(),
	}
	err := Publish("", "stats", rsp /* , tagBase */)
	if err != nil {
		counters.Errors.Inc()
		slog.Error(tagBase.F("Error"), "err", err)
	}
}

var SendStatus workers.OnlineStatusChangeHandler = func(
	target string,
	status workers.OnlineStatus,
	lastSeen time.Time,
	updSource workers.UpdSource,
	// tag utils.Tag,
) {
	rsp := StatusResponse{
		Status:    status,
		LastSeen:  lastSeen,
		UpdSource: updSource,
	}
	err := Publish(target, "status", rsp /* , tag */)
	if err != nil {
		counters.Errors.Inc()
		slog.Error(tagBase.F("Error"), "err", err)
	}
}

func SendOpFeedback(req *InMessage, target string, message string, isError bool /* , tag utils.Tag */) {
	rsp := SequencedResponse{
		Message: message,
		IsError: isError,
		Seq:     req.Seq,
	}
	if isError {
		counters.Errors.Inc()
		slog.Error(tagBase.F("Error"), "err", message)
	}
	err := Publish(target, "rsp", rsp /* , tag */)
	if err != nil {
		counters.Errors.Inc()
		slog.Error(tagBase.F("Error"), "err", err)
	}
}

func handleAction(action string, target string, message *InMessage) {
	handled := false
	switch action {
	case "get-stats":
		SendStats()
		handled = true
	case "get":
		slog.Debug(tagBase.F("Getting status for %v", target))
		worker, err := workers.Get(target)
		if err == nil {
			SendStatus(target, worker.Status(), worker.LastSeen(), workers.UPD_SOURCE_MQTT_GET /* , worker.Tag() */)
			handled = true
		} else {
			SendOpFeedback(message, target, err.Error(), true /* , worker.Tag() */)
		}
	case "add":
		slog.Debug(tagBase.F("Adding new worker for %v", target))
		_, err := workers.Create(
			target,
			SendStatus,
		)
		if err == nil {
			SendOpFeedback(message, target, "added", false /* , tagBase */)
			SendStats()
			handled = true
		} else {
			SendOpFeedback(message, target, err.Error(), true /* , tagBase */)
		}
	case "del":
		slog.Debug(tagBase.F("Deleting worker for %v", target))
		err := workers.Delete(target, SendStatus)
		if err == nil {
			SendOpFeedback(message, target, "deleted", false /* , tagBase */)
			SendStats()
			handled = true
		} else {
			SendOpFeedback(message, target, err.Error(), true /* , tagBase */)
		}
	}
	if handled {
		counters.ActionsHandled.WithLabelValues(action, target).Inc()
	}
}

var defaultMessageHandler MqttLib.MessageHandler = func(client MqttLib.Client, msg MqttLib.Message) {

	counters.MqttReceived.Inc()

	topic := msg.Topic()

	// fmt.Println( /* string(msg.Payload()) */ `"foo"`)
	// slog.Debug(`"foo"`, "foo", `"foo"`)
	// fmt.Println(fmt.Sprint(string(msg.Payload())))
	// "foo".MarshalText()

	slog.Debug(
		tagBase.F("Received"),
		"topic", topic,
		"payload", utils_local.Truncate(string(msg.Payload()), 80),
	)

	tt := strings.Split(topic, "/")

	ttlen := len(tt)

	message := InMessage{}

	if len(msg.Payload()) > 0 && msg.Payload()[0] == LBRACKET {
		err := json.Unmarshal(msg.Payload(), &message)
		if err != nil {
			counters.Errors.Inc()
			slog.Error(tagBase.F("Error"), "err", err)
		}
	}

	if ttlen == 2 {
		action := tt[1]
		handleAction(action, "", &message)
	} else if ttlen == 3 {
		target := tt[1]
		action := tt[2]
		handleAction(action, target, &message)
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

func subscribe(client MqttLib.Client, topic string, wg *sync.WaitGroup) {
	defer wg.Done()
	if token := client.Subscribe(topic, 0, nil); token.Wait() && token.Error() != nil {
		counters.Errors.Inc()
		slog.Error(tagBase.F("client.Subscribe()"), "err", token.Error())
	}
	slog.Info(tagBase.F("Subscribed"), "topic", topic)
}

func subscribeAll(client MqttLib.Client) {
	var wg sync.WaitGroup

	topics := []string{
		fmt.Sprintf("%s/get-stats", registry.Config.MqttTopicBase),
		fmt.Sprintf("%s/+/add", registry.Config.MqttTopicBase),
		fmt.Sprintf("%s/+/get", registry.Config.MqttTopicBase),
		fmt.Sprintf("%s/+/del", registry.Config.MqttTopicBase),
	}

	for _, topic := range topics {
		wg.Add(1)
		go subscribe(client, topic, &wg)
	}
	wg.Wait()
	slog.Debug(tagBase.F("All subscribtions are settled"))
}
