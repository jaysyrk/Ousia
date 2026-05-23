package observability

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	RequestsTotal	= prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:	"ousia_requests_total",
			Help:	"Total number of requests processed by the gateway.",
		},
		[]string{"method", "host", "upstream", "status"},
	)

	RequestDuration	= prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:		"ousia_request_duration_ms",
			Help:		"Request duration in milliseconds.",
			Buckets:	[]float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000},
		},
		[]string{"method", "host", "upstream"},
	)

	ActiveConnections	= prometheus.NewGauge(prometheus.GaugeOpts{
		Name:	"ousia_active_connections",
		Help:	"Number of active connections being handled.",
	})

	HealthyEndpoints	= prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:	"ousia_healthy_endpoints",
			Help:	"Number of healthy endpoints per upstream pool.",
		},
		[]string{"pool"},
	)

	MeshRequestsTotal	= prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name:	"ousia_mesh_requests_total",
			Help:	"Total number of service-to-service requests in the mesh.",
		},
		[]string{"source", "destination", "status", "method"},
	)

	MeshRequestDuration	= prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:		"ousia_mesh_request_duration_ms",
			Help:		"Service-to-service request duration in milliseconds.",
			Buckets:	[]float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000},
		},
		[]string{"source", "destination", "method"},
	)
)

func InitMetrics() {
	prometheus.MustRegister(RequestsTotal)
	prometheus.MustRegister(RequestDuration)
	prometheus.MustRegister(ActiveConnections)
	prometheus.MustRegister(HealthyEndpoints)
	prometheus.MustRegister(MeshRequestsTotal)
	prometheus.MustRegister(MeshRequestDuration)
}

func StartAdminServer(addr string, register func(*http.ServeMux)) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})

	if register != nil {
		register(mux)
	}

	go func() {
		fmt.Printf("Ousia admin server listening on %s\n", addr)
		if err := http.ListenAndServe(addr, mux); err != nil {
			fmt.Printf("admin server error: %v\n", err)
		}
	}()
}

func GetStatsJSON() map[string]interface{} {
	mfs, _ := prometheus.DefaultGatherer.Gather()
	res := make(map[string]interface{})
	for _, mf := range mfs {
		if mf.Name == nil {
			continue
		}

		var metricsList []map[string]interface{}
		for _, m := range mf.Metric {
			metricData := make(map[string]interface{})
			labels := make(map[string]string)
			for _, l := range m.Label {
				labels[*l.Name] = *l.Value
			}
			if len(labels) > 0 {
				metricData["labels"] = labels
			}
			if m.Counter != nil {
				metricData["value"] = m.Counter.GetValue()
			} else if m.Gauge != nil {
				metricData["value"] = m.Gauge.GetValue()
			} else if m.Histogram != nil {
				metricData["count"] = m.Histogram.GetSampleCount()
				metricData["sum"] = m.Histogram.GetSampleSum()
			}
			metricsList = append(metricsList, metricData)
		}
		res[*mf.Name] = metricsList
	}
	return res
}
