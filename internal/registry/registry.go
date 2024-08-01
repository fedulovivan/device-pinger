package registry

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
	Config    ConfigStorage
	startTime time.Time
)

type Uptime struct {
	time.Duration
}

func (d Uptime) MarshalJSON() (b []byte, err error) {
	return []byte(fmt.Sprintf(`"%s"`, d.String())), nil
}

type ConfigStorage struct {
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
func GetExpectedEnvVars() []string {
	typ := reflect.TypeOf(ConfigStorage{})
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

func init() {
	fileName, withConf := os.LookupEnv("CONF")
	if !withConf {
		fileName = ".env"
	}
	err := godotenv.Load(fileName)
	if err != nil {
		fmt.Println("godotenv.Load()", err)
	} else {
		fmt.Println("env variables were loaded from file", fileName)
	}
	if err := envconfig.Process(context.Background(), &Config); err != nil {
		panic("failed loading env variables into struct: " + err.Error())
	}
	fmt.Printf("starting with config %+v\n", Config)
	if Config.IsDev {
		fmt.Println("all known config variables", GetExpectedEnvVars())
	}
}

func RecordStartTime() {
	if !startTime.IsZero() {
		panic("expected to be called only once")
	}
	startTime = time.Now()
}

func GetUptime() Uptime {
	return Uptime{time.Since(startTime)}
}
