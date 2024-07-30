package mqtt

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/fedulovivan/device-pinger/internal/config"
	"github.com/fedulovivan/device-pinger/internal/utils"
	"github.com/fedulovivan/device-pinger/internal/workers"

	MqttLib "github.com/eclipse/paho.mqtt.golang"
)

type SequencedRequest struct {
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
	Workers     int           `json:"workers"`
	MemoryAlloc uint64        `json:"memoryAlloc"`
	Uptime      config.Uptime `json:"uptime"`
}

var client MqttLib.Client

func GetBokerString() string {
	cfg := config.GetInstance()
	return fmt.Sprintf("tcp://%s:%d", cfg.MqttHost, cfg.MqttPort)
}

func Shutdown() {
	slog.Info("[MQTT] Shutdown")
	client.Disconnect(250)
}

func Init() *MqttLib.Client {
	cfg := config.GetInstance()
	opts := MqttLib.NewClientOptions()
	opts.AddBroker(GetBokerString())
	opts.SetClientID(cfg.MqttClientId)
	opts.SetUsername(cfg.MqttUsername)
	opts.SetPassword(cfg.MqttPassword)
	opts.SetDefaultPublishHandler(defaultMessageHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	client = MqttLib.NewClient(opts)
	slog.Info("[MQTT] Connecting...")
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		slog.Error("[MQTT]", "error", token.Error())
	} else {
		subscribeAll(client)
	}
	return &client
}

func BuildTopic(target string, action string) string {
	cfg := config.GetInstance()
	parts := []string{cfg.MqttTopicBase}
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
		Workers:     workers.GetCount(),
		Uptime:      config.GetUptime(),
	}
	err := Publish("", "stats", rsp)
	if err != nil {
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
		slog.Error("[MQTT]", "err", err)
	}
}

func SendOpFeedback(req *SequencedRequest, target string, message string, isError bool) {
	rsp := SequencedResponse{
		Message: message,
		IsError: isError,
		Seq:     req.Seq,
	}
	err := Publish(target, "rsp", rsp)
	if err != nil {
		slog.Error("[MQTT]", "err", err)
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

	if ttlen == 2 {

		action := tt[1]

		switch action {
		case "get-stats":
			SendStats()
		}

	} else if ttlen == 3 {

		target := tt[1]
		action := tt[2]
		exist := workers.Has(target)

		req := SequencedRequest{}
		if len(msg.Payload()) > 0 && msg.Payload()[0] == 123 /* 123 = "{" */ {
			err := json.Unmarshal(msg.Payload(), &req)
			if err != nil {
				slog.Error("[MQTT]", "err", err)
			}
		}

		switch action {
		case "get":
			slog.Debug("[MQTT] Getting status for " + target)
			if exist {
				worker := workers.Get(target)
				SendStatus(target, worker.Status(), worker.LastSeen(), workers.UPD_SOURCE_MQTT_GET)
			} else {
				SendOpFeedback(&req, target, "not exist", true)
			}
		case "add":
			slog.Debug("[MQTT] Adding new worker for " + target)
			if exist {
				SendOpFeedback(&req, target, "already exist", true)
			} else {
				workers.Add(workers.Create(
					target,
					SendStatus,
				))
				SendOpFeedback(&req, target, "added", false)
				SendStats()
				utils.PrintMemUsage()
			}
		case "del":
			slog.Debug("[MQTT] Deleting worker for " + target)
			if exist {
				workers.Delete(target, SendStatus)
				SendOpFeedback(&req, target, "deleted", false)
				SendStats()
				utils.PrintMemUsage()
			} else {
				SendOpFeedback(&req, target, "not exist", true)
			}
		}

	} else {
		slog.Error("[MQTT] Unexpected topic format " + topic)
		return
	}

}

var connectHandler MqttLib.OnConnectHandler = func(client MqttLib.Client) {
	slog.Info("[MQTT] Connected", "broker", GetBokerString())
}

var connectLostHandler MqttLib.ConnectionLostHandler = func(client MqttLib.Client, err error) {
	slog.Error("[MQTT] Connection lost", "err", err)
}

func subscribe(client MqttLib.Client, topic string, wg *sync.WaitGroup) {
	defer wg.Done()
	if token := client.Subscribe(topic, 0, nil); token.Wait() && token.Error() != nil {
		slog.Error("[MQTT] client.Subscribe()", "err", token.Error())
	}
	slog.Info("[MQTT] Subscribed", "topic", topic)
}

func subscribeAll(client MqttLib.Client) {
	cfg := config.GetInstance()
	var wg sync.WaitGroup
	for _, topic := range []string{cfg.MqttTopicBase + "/#"} {
		wg.Add(1)
		go subscribe(client, topic, &wg)
	}
	wg.Wait()
	slog.Info("[MQTT] All subscribtions are settled")
}
