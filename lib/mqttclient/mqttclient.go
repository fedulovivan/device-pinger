package mqttclient

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/fedulovivan/device-pinger/lib/config"
	"github.com/fedulovivan/device-pinger/lib/utils"
	"github.com/fedulovivan/device-pinger/lib/workers"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var client mqtt.Client

func GetBokerString() string {
	cfg := config.GetInstance()
	return fmt.Sprintf("tcp://%s:%d", cfg.MqttBroker, cfg.MqttPort)
}

func Shutdown() {
	log.Println("[MQTT] Shutdown")
	client.Disconnect(250)
}

func Init() mqtt.Client {
	cfg := config.GetInstance()
	opts := mqtt.NewClientOptions()
	opts.AddBroker(GetBokerString())
	opts.SetClientID(cfg.MqttClientId)
	opts.SetUsername(cfg.MqttUsername)
	opts.SetPassword(cfg.MqttPassword)
	opts.SetDefaultPublishHandler(defaultMessageHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	client = mqtt.NewClient(opts)
	log.Println("[MQTT]", "Connecting to broker...")
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Println("[MQTT]", token.Error())
	} else {
		subscribeAll(client)
	}
	return client
}

var HandleOnlineChange workers.OnlineStatusChangeHandler = func(target string, online bool) {
	token := client.Publish(
		fmt.Sprintf("device-pinger/%v/status", target),
		0,
		false,
		fmt.Sprintf(`{"online":%v}`, online),
	)
	token.Wait()
}

var defaultMessageHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {

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

	switch action {
	case "add":
		log.Println("[MQTT] Adding new worker for " + target)
		if workers.Has(target) {
			log.Println("[MQTT] Already exist")
		} else {
			workers.Push(workers.Create(
				target,
				HandleOnlineChange,
			))
		}
	case "del":
		log.Println("[MQTT] Deleting worker for " + target)
		if workers.Has(target) {
			workers.Delete(target)
		} else {
			log.Println("[MQTT] Not yet exist")
		}
	}
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	log.Printf("[MQTT] Connected to %v\n", GetBokerString())
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	log.Printf("[MQTT] Connection lost: %v\n", err)
}

func subscribe(client mqtt.Client, topic string, wg *sync.WaitGroup) {
	defer wg.Done()
	if token := client.Subscribe(topic, 0, nil); token.Wait() && token.Error() != nil {
		log.Fatalf("[MQTT] client.Subscribe() %v", token.Error())
	}
	log.Printf("[MQTT] Subscribed to %s topic\n", topic)
}

func subscribeAll(client mqtt.Client) {
	cfg := config.GetInstance()
	var wg sync.WaitGroup
	for _, topic := range []string{cfg.MqttTopicBase + "/#"} {
		wg.Add(1)
		go subscribe(client, topic, &wg)
	}
	wg.Wait()
	log.Printf("[MQTT] All subscribtions are settled\n")
}
