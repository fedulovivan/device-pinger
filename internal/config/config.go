package config

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/sethvargo/go-envconfig"
)

var (
	cfg       Config
	loaded    bool
	startTime time.Time
)

type Uptime struct {
	time.Duration
}

func (d Uptime) MarshalJSON() (b []byte, err error) {
	return []byte(fmt.Sprintf(`"%s"`, d.String())), nil
}

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
	IsDev                  bool          `env:"DEV,default=false"`
}

// use reflection to parse Config struct tags and report unexpected variables from .env file
func GetKnown() []string {
	typ := reflect.TypeOf(Config{})
	var m []string
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if tagValue := field.Tag.Get("env"); tagValue != "" {
			tt := strings.Split(tagValue, ",")
			if len(tt) > 0 {
				m = append(m, tt[0] /* field.Name */)
			}
		}
	}
	return m
}

func load() {
	fileName, withConf := os.LookupEnv("CONF")
	if !withConf {
		fileName = ".env"
	}
	err := godotenv.Load(fileName)
	if err != nil {
		/* slog.Warn */ fmt.Println( /* [MAIN]  */ "godotenv.Load()", "err", err)
	} else {
		/* slog.Info */ fmt.Println( /* [MAIN]  */ "env variables were loaded", "file", fileName)
	}
	ctx := context.Background()
	if err := envconfig.Process(ctx, &cfg); err != nil {
		/* slog.Error */ fmt.Println( /* [MAIN]  */ "failed loading env variables into config struct", "err", err)
		os.Exit(1)
	}
	/* slog.Info */ fmt.Printf( /* [MAIN]  */ "starting with config %+v\n", cfg)
	loaded = true

	if cfg.IsDev {
		fmt.Println("all known config variables", GetKnown())
	}
}

func SetStartTime() {
	if !startTime.IsZero() {
		panic("Expected to be called only once")
	}
	startTime = time.Now()
}

func GetUptime() Uptime {
	return Uptime{time.Since(startTime)}
}

func GetInstance() *Config {
	if !loaded {
		load()
	}
	return &cfg
}
