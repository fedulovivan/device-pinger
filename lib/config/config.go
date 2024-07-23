package config

import (
	"context"
	"log"
	"time"

	"github.com/joho/godotenv"
	"github.com/sethvargo/go-envconfig"
)

var cfg Config

var loaded bool

type Config struct {
	TargetIps            []string      `env:"TARGET_IPS,required"`
	MqttBroker           string        `env:"MQTT_BROKER,required"`
	MqttUsername         string        `env:"MQTT_USERNAME"`
	MqttPassword         string        `env:"MQTT_PASSWORD"`
	MqttPort             int           `env:"MQTT_PORT,default=1883"`
	MqttTopicBase        string        `env:"MQTT_TOPIC_BASE,default=device-pinger"`
	MqttClientId         string        `env:"MQTT_CLIENT_ID,default=device-pinger"`
	OfflineAfter         time.Duration `env:"OFFLINE_AFTER,default=30s"`
	PingerInterval       time.Duration `env:"PINGER_INTERVAL,default=5s"`
	OfflineCheckInterval time.Duration `env:"OFFLINE_CHECK_INTERVAL,default=5s"`
}

func load() {
	godotenv.Load()
	ctx := context.Background()
	if err := envconfig.Process(ctx, &cfg); err != nil {
		log.Fatal(err)
	}
	log.Printf("Starting with config %+v\n", cfg)
	loaded = true
}

func GetInstance() *Config {
	if !loaded {
		load()
	}
	return &cfg
}
