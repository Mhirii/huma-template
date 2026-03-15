package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	_ "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	_ "github.com/prometheus/client_golang/prometheus/promauto"
	_ "github.com/prometheus/client_golang/prometheus/promhttp"
)

type metrics struct {
	OpsProcessed        *prometheus.Counter
	HttpRequestsSeconds *prometheus.HistogramVec
	HttpRequestsTotal   *prometheus.CounterVec
	Uptime              prometheus.Gauge
}

var Metrics *metrics

var Reg *prometheus.Registry

func InitMetrics() *prometheus.Registry {
	Reg = prometheus.NewRegistry()
	HTTPRequestsSeconds := promauto.With(Reg).NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Time (in seconds) spent serving HTTP requests",
		Buckets: prometheus.DefBuckets,
	}, []string{"status", "method", "operation", "path"})
	HTTPRequestsTotal := promauto.With(Reg).NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests",
	}, []string{"status", "method", "operation", "path"})
	Uptime := promauto.With(Reg).NewGauge(prometheus.GaugeOpts{
		Name: "process_uptime_seconds",
		Help: "Time (in seconds) since the process started",
	})
	Metrics = &metrics{
		HttpRequestsSeconds: HTTPRequestsSeconds,
		HttpRequestsTotal:   HTTPRequestsTotal,
		Uptime:              Uptime,
	}
	go func() {
		for {
			Metrics.Uptime.Add(2)
			time.Sleep(2 * time.Second)
		}
	}()
	return Reg
}

func GetMetrics(reg prometheus.Registerer) *metrics {
	return Metrics
}
func GetReg() *prometheus.Registry {
	return Reg
}
