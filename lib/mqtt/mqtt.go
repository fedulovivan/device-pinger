package mqtt

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/fedulovivan/device-pinger/lib/config"
	"github.com/fedulovivan/device-pinger/lib/utils"
	"github.com/fedulovivan/device-pinger/lib/workers"

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
	Status workers.OnlineStatus `json:"status"`
}

var client MqttLib.Client

func GetBokerString() string {
	cfg := config.GetInstance()
	return fmt.Sprintf("tcp://%s:%d", cfg.MqttHost, cfg.MqttPort)
}

func Shutdown() {
	log.Println("[MQTT] Shutdown")
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
	log.Println("[MQTT]", "Connecting to broker...")
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Println("[MQTT]", token.Error())
	} else {
		subscribeAll(client)
	}
	return &client
}

func GetTopic(target string, action string) string {
	cfg := config.GetInstance()
	return strings.Join([]string{cfg.MqttTopicBase, target, action}, "/")
}

func Publish(target string, action string, msg any) error {
	resString, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	token := client.Publish(
		GetTopic(target, action),
		0,
		false,
		resString,
	)
	token.Wait()
	return nil
}

var SendStatus workers.OnlineStatusChangeHandler = func(target string, status workers.OnlineStatus) {
	msg := StatusResponse{status}
	Publish(target, "status", msg)
}

func SendOpFeedback(req *SequencedRequest, target string, message string, isError bool) {
	msg := SequencedResponse{
		Message: message,
		IsError: isError,
		Seq:     req.Seq,
	}
	Publish(target, "rsp", msg)
}

var defaultMessageHandler MqttLib.MessageHandler = func(client MqttLib.Client, msg MqttLib.Message) {
	topic := msg.Topic()

	log.Printf(
		"[MQTT] TOPIC=%s MESSAGE=%s\n",
		topic,
		utils.Truncate(string(msg.Payload()), 80),
	)

	tt := strings.Split(topic, "/")

	if len(tt) != 3 {
		log.Println("[MQTT] Unexpected format for topic " + topic)
		return
	}

	target := tt[1]
	action := tt[2]
	exist := workers.Has(target)

	req := SequencedRequest{}
	json.Unmarshal(msg.Payload(), &req)

	switch action {
	case "get":
		log.Println("[MQTT] Getting status for " + target)
		if exist {
			worker := workers.Get(target)
			SendStatus(target, worker.Status())
		} else {
			SendOpFeedback(&req, target, "not exist", true)
		}
	case "add":
		log.Println("[MQTT] Adding new worker for " + target)
		if exist {
			SendOpFeedback(&req, target, "already exist", true)
		} else {
			workers.Add(workers.Create(
				target,
				SendStatus,
			))
			SendOpFeedback(&req, target, "added", false)
		}
	case "del":
		log.Println("[MQTT] Deleting worker for " + target)
		if exist {
			workers.Delete(target, SendStatus)
			SendOpFeedback(&req, target, "deleted", false)
		} else {
			SendOpFeedback(&req, target, "not exist", true)
		}
	}
}

var connectHandler MqttLib.OnConnectHandler = func(client MqttLib.Client) {
	log.Printf("[MQTT] Connected to %v\n", GetBokerString())
}

var connectLostHandler MqttLib.ConnectionLostHandler = func(client MqttLib.Client, err error) {
	log.Printf("[MQTT] Connection lost: %v\n", err)
}

func subscribe(client MqttLib.Client, topic string, wg *sync.WaitGroup) {
	defer wg.Done()
	if token := client.Subscribe(topic, 0, nil); token.Wait() && token.Error() != nil {
		log.Fatalf("[MQTT] client.Subscribe() %v", token.Error())
	}
	log.Printf("[MQTT] Subscribed to %s topic\n", topic)
}

func subscribeAll(client MqttLib.Client) {
	cfg := config.GetInstance()
	var wg sync.WaitGroup
	for _, topic := range []string{cfg.MqttTopicBase + "/#"} {
		wg.Add(1)
		go subscribe(client, topic, &wg)
	}
	wg.Wait()
	log.Printf("[MQTT] All subscribtions are settled\n")
}
