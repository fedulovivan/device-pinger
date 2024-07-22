package constants

import "time"

const (
	MQTT_BROKER    = "192.168.88.188"
	MQTT_PORT      = 1883
	MQTT_CLIENT_ID = "device-pinger"
	MQTT_USERNAME  = "mosquitto"
	MQTT_PASSWORD  = "5Ysm3jAsVP73nva"
	MQTT_TOPIC     = "device-pinger/#"
)

var TARGET_IPS = []string{

	// IPHONE_15_PRO_IP in "АП"
	"192.168.0.11",
	// IPHONE_14_IP
	"192.168.88.62",
	// IPHONE_15_PRO_IP in "Богородского"
	"192.168.88.71",
	// PIXEL_5_IP
	"192.168.88.68",
	// Router in "АП"
	"192.168.0.1",

	// "foo-bar",
	// "255.255.255.255",
}

const (
	// if no responses were received after this period device is considered offline
	OFFLINE_AFTER = time.Second * 30
	// how often offlne checking ticker is executed
	OFFLINE_CHECK_INTERVAL = time.Second * 5
	// how often ping requests are sent
	PINGER_INTERVAL = time.Second * 5
)
