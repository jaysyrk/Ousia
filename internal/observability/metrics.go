package observability

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	RequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ousia_requests_total",
			Help: "Total number of requests processed by the gateway.",
		},
		[]string{"method", "host", "upstream", "status"},
	)

	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ousia_request_duration_ms",
			Help:    "Request duration in milliseconds.",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000},
		},
		[]string{"method", "host", "upstream"},
	)

	ActiveConnections = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "ousia_active_connections",
		Help: "Number of active connections being handled.",
	})

	HealthyEndpoints = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ousia_healthy_endpoints",
			Help: "Number of healthy endpoints per upstream pool.",
		},
		[]string{"pool"},
	)
)

func InitMetrics() {
	prometheus.MustRegister(RequestsTotal)
	prometheus.MustRegister(RequestDuration)
	prometheus.MustRegister(ActiveConnections)
	prometheus.MustRegister(HealthyEndpoints)
}

func StartAdminServer(addr string) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})

	go func() {
		fmt.Printf("Ousia admin server listening on %s\n", addr)
		if err := http.ListenAndServe(addr, mux); err != nil {
			fmt.Printf("admin server error: %v\n", err)
		}
	}()
}
