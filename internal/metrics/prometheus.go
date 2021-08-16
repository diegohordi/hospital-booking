// Package metrics contains middlewares and counters for metrics gathering.
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
)

// HTTP Requests total counter
var totalRequests = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "HTTP Requests.",
	},
	[]string{"path"},
)

// HTTP Response status
var duration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name: "http_duration",
		Help: "HTTP Requests Duration",
	},
	[]string{"path"},
)

func init() {
	err := prometheus.Register(totalRequests)
	if err != nil {
		panic(err)
	}
	err = prometheus.Register(duration)
	if err != nil {
		panic(err)
	}
}

// PrometheusMiddleware instruments the given request and register metrics.
func PrometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timer := prometheus.NewTimer(duration.WithLabelValues(r.RequestURI))
		next.ServeHTTP(w, r)
		totalRequests.WithLabelValues(r.RequestURI).Inc()
		timer.ObserveDuration()
	})
}
