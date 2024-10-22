package counters

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var ApiRequests = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "pinger_api_requests",
	},
	[]string{"topic"},
)

var Errors = promauto.NewCounter(
	prometheus.CounterOpts{
		Name: "pinger_errors",
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
