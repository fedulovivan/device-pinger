package mqtt

// mqtt topics
// pub device-pinger/<ip>/status {"online":true}
// sub device-pinger/<ip>/add {} or nil
// sub device-pinger/<ip>/remove {} or nil

import (
	"fmt"

	"device-pinger/constants"
	"device-pinger/utils"
	"log"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var client mqtt.Client

func Shutdown() {
	log.Println("[MQTT] Shutdown")
	client.Disconnect(250)
}

func Client() mqtt.Client {
	return client
}

func Init() mqtt.Client {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", constants.MQTT_BROKER, constants.MQTT_PORT))
	opts.SetClientID(constants.MQTT_CLIENT_ID)
	opts.SetUsername(constants.MQTT_USERNAME)
	opts.SetPassword(constants.MQTT_PASSWORD)
	opts.SetDefaultPublishHandler(defaultMessageHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	client = mqtt.NewClient(opts)
	log.Println("[MQTT]", "Connecting to broker...")
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Println("[MQTT]", token.Error())
	} else {
		// log.Println("[MQTT]", "Connected 1")
		subscribeAll(client)
	}
	return client
}

var defaultMessageHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {

	log.Printf(
		"[MQTT]->\nTOPIC: %s\nMESSAGE: %s\n",
		msg.Topic(),
		utils.Truncate(msg.Payload(), 80),
	)

	// tt := strings.Split(msg.Topic(), "/")
	// if len(tt) < 2 {
	// 	return
	// }
	// if zigbeeDeviceId := tt[1]; strings.HasPrefix(zigbeeDeviceId, "0x") {

	// 	log.Printf("[MQTT] zigbee device id is %v\n", zigbeeDeviceId)

	// 	var m models.AnyJson
	// 	if err := json.Unmarshal(msg.Payload(), &m); err != nil {
	// 		log.Printf("Failed to parse json value\n%s\n", msg.Payload())
	// 	} else {
	// 		models.PutLastDeviceMessage(models.LastDeviceMessage{
	// 			/* DeviceId */ zigbeeDeviceId,
	// 			/* Message */ m,
	// 			/* Timestamp */ time.Now(),
	// 			/* Name */ constants.DEVICE_NAME[zigbeeDeviceId],
	// 		})
	// 	}

	// }
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
	for _, topic := range []string{constants.MQTT_TOPIC} {
		wg.Add(1)
		go subscribe(client, topic, &wg)
	}
	wg.Wait()
	log.Printf("[MQTT] All subscribtions are settled\n")
}
