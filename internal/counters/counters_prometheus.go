package counters

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var ActionsHandled = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "pinger_actions_handled",
	},
	[]string{"action", "target"},
)

var Errors = promauto.NewCounter(
	prometheus.CounterOpts{
		Name: "pinger_errors",
	},
)

var MqttReceived = promauto.NewCounter(
	prometheus.CounterOpts{
		Name: "pinger_mqtt_received",
	},
)

var MqttPublished = promauto.NewCounter(
	prometheus.CounterOpts{
		Name: "pinger_mqtt_published",
	},
)

var Workers = promauto.NewGauge(
	prometheus.GaugeOpts{
		Name: "pinger_workers",
	},
)

var Uptime = promauto.NewGauge(
	prometheus.GaugeOpts{
		Name: "pinger_uptime",
	},
)
