package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	MetricBuildInfo = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ldap_sd_build_info",
			Help: "Build information prometheus-ldap-sd ",
		},
		[]string{"version", "git_hash"},
	)
	MetricServerRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ldap_sd_server_requests_total",
			Help: "Total number of requests to the remote LDAP server",
		},
		[]string{"group_name"},
	)
	MetricServerRequestsFailed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ldap_sd_server_requests_failed_total",
			Help: "Total number of requests to the remote LDAP server which have failed",
		},
		[]string{"group_name"},
	)
	MetricRequestsFromCache = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ldap_sd_cache_hit_total",
			Help: "Number of requests served directly from local cache",
		},
		[]string{"group_name"},
	)
	MetricCacheUpdateSuccess = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ldap_sd_cache_update_success_total",
			Help: "Number of updates to the cache which have failed",
		},
		[]string{"group_name"},
	)
	MetricCacheUpdateFail = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ldap_sd_cache_update_fail_total",
			Help: "Number of updates to the cache which have succeeded",
		},
		[]string{"group_name"},
	)
	MetricReconnect = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "ldap_sd_connect_total",
			Help: "Number of times the connection to remote LDAP server was re-connected.",
		},
	)
	MetricGroupNumObjects = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ldap_sd_target_group_num_objects",
			Help: "Number of objects last discovered for the target group.",
		},
		[]string{"group_name"},
	)
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func NewResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// func InitCounter(metric *prometheus.CounterVec, targetGroup string) {
// 	metric.WithLabelValues(targetGroup)
// }
