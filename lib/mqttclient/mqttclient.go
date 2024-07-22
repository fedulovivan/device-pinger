package mqttclient

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/fedulovivan/device-pinger/lib/utils"
	"github.com/fedulovivan/device-pinger/lib/workers"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var client mqtt.Client

var brokerString = fmt.Sprintf(
	"tcp://%s:%s",
	os.Getenv("MQTT_BROKER"),
	os.Getenv("MQTT_PORT"),
)

func Shutdown() {
	log.Println("[MQTT] Shutdown")
	client.Disconnect(250)
}

func Init() mqtt.Client {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerString)
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
	log.Printf("[MQTT] Connected to %v\n", brokerString)
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
	var wg sync.WaitGroup
	for _, topic := range []string{os.Getenv("MQTT_TOPIC_BASE") + "/#"} {
		wg.Add(1)
		go subscribe(client, topic, &wg)
	}
	wg.Wait()
	log.Printf("[MQTT] All subscribtions are settled\n")
}
