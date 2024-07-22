package mqttclient

// mqtt topics
// PUB device-pinger/<ip>/status, payload {"online":true}
// SUB device-pinger/<ip>/add, any payload
// SUB device-pinger/<ip>/remove, any payload

import (
	"device-pinger/lib/utils"
	"device-pinger/lib/workers"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var client mqtt.Client

func Shutdown() {
	log.Println("[MQTT] Shutdown")
	client.Disconnect(250)
}

func Init() mqtt.Client {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf(
		"tcp://%s:%s",
		os.Getenv("MQTT_BROKER"),
		os.Getenv("MQTT_PORT"),
	))
	opts.SetClientID(os.Getenv("MQTT_CLIENT_ID"))
	opts.SetUsername(os.Getenv("MQTT_USERNAME"))
	opts.SetPassword(os.Getenv("MQTT_PASSWORD"))
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
		utils.Truncate(msg.Payload(), 80),
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
	case "delete":
		log.Println("[MQTT] Deleting worker for " + target)
		if workers.Has(target) {
			workers.Delete(target)
		} else {
			log.Println("[MQTT] Not yet exist")
		}
	}
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	log.Printf("[MQTT] Connected\n")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	log.Printf("[MQTT] Connection lost: %v\n", err)
}

func subscribe(client mqtt.Client, topic string, wg *sync.WaitGroup) {
	defer wg.Done()
	if token := client.Subscribe(topic, 0, nil); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}
	log.Printf("[MQTT] Subscribed to %s topic\n", topic)
}

func subscribeAll(client mqtt.Client) {
	var wg sync.WaitGroup
	for _, topic := range []string{os.Getenv("MQTT_TOPIC_BASE") + "/#"} {
		wg.Add(1)
		go subscribe(client, topic, &wg)
	}
	wg.Wait()
	log.Printf("[MQTT] All subscribtions are settled\n")
}
