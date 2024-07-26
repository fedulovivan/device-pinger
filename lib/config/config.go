package config

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/sethvargo/go-envconfig"
)

var (
	cfg    Config
	loaded bool
)

type Config struct {
	TargetIps              []string      `env:"TARGET_IPS"`
	MqttHost               string        `env:"MQTT_HOST,default=mosquitto"`
	MqttPort               int           `env:"MQTT_PORT,default=1883"`
	MqttUsername           string        `env:"MQTT_USERNAME"`
	MqttPassword           string        `env:"MQTT_PASSWORD"`
	MqttTopicBase          string        `env:"MQTT_TOPIC_BASE,default=device-pinger"`
	MqttClientId           string        `env:"MQTT_CLIENT_ID,default=device-pinger"`
	OfflineAfter           time.Duration `env:"OFFLINE_AFTER,default=30s"`
	PingerInterval         time.Duration `env:"PINGER_INTERVAL,default=5s"`
	OfflineCheckInterval   time.Duration `env:"OFFLINE_CHECK_INTERVAL,default=5s"`
	PeriodicUpdateInterval time.Duration `env:"PERIODIC_UPDATE_INTERVAL,default=10m"`
	LogLevel               slog.Level    `env:"LOG_LEVEL,default=debug"`
}

func load() {
	fileName, withConf := os.LookupEnv("CONF")
	if !withConf {
		fileName = ".env"
	}
	err := godotenv.Load(fileName)
	if err != nil {
		slog.Error("[MAIN] godotenv.Load()", "err", err)
	} else {
		slog.Info("[MAIN] env variables were loaded", "file", fileName)
	}
	ctx := context.Background()
	if err := envconfig.Process(ctx, &cfg); err != nil {
		slog.Error("[MAIN] failed loading env variables into config struct", "err", err)
		os.Exit(1)
	}
	slog.Info("[MAIN] Starting with", "config", cfg)
	loaded = true
}

func GetInstance() *Config {
	if !loaded {
		load()
	}
	return &cfg
}
