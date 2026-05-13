package observability

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// metrics.go provides Prometheus metrics for Redis operations and cache performance.
// Metrics are registered globally and exported via /metrics endpoint (added in app.go).

var (
	// RedisJSONSetLatency measures latency (milliseconds) of JSON.SET operations.
	RedisJSONSetLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "redis_json_set_latency_ms",
			Help:    "Latency of RedisJSON SET operations in milliseconds",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500},
		},
		[]string{"status"}, // status: "success" or "error"
	)

	// RedisJSONGetLatency measures latency (milliseconds) of JSON.GET operations.
	RedisJSONGetLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "redis_json_get_latency_ms",
			Help:    "Latency of RedisJSON GET operations in milliseconds",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500},
		},
		[]string{"status"},
	)

	// CacheHits counts total cache hits in the item cache decorator.
	CacheHits = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Total number of cache hits in item cache",
		},
	)

	// CacheMisses counts total cache misses in the item cache decorator.
	CacheMisses = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "Total number of cache misses in item cache",
		},
	)

	// StreamProcessingLatency measures end-to-end audit event processing time.
	StreamProcessingLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "stream_processing_latency_ms",
			Help:    "Latency of stream event processing in milliseconds",
			Buckets: []float64{5, 10, 25, 50, 100, 250, 500, 1000},
		},
		[]string{"status"}, // status: "success" or "error"
	)

	// StreamEventsProcessed counts total events processed by the audit worker.
	StreamEventsProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "stream_events_processed_total",
			Help: "Total stream events processed by audit worker",
		},
		[]string{"action"}, // action: "created", "updated", "deleted"
	)

	// BloomFilterChecks tracks Bloom filter existence checks.
	BloomFilterChecks = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bloom_filter_checks_total",
			Help: "Total Bloom filter existence checks",
		},
		[]string{"result"}, // result: "exists" or "not_exists"
	)

	// RediSearchLatency measures full-text search query latency.
	RediSearchLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "redis_search_latency_ms",
			Help:    "Latency of RediSearch FT.SEARCH operations in milliseconds",
			Buckets: []float64{5, 10, 25, 50, 100, 250, 500},
		},
		[]string{"status"},
	)
)

// RecordJSONSetLatency records the latency of a JSON.SET operation.
func RecordJSONSetLatency(start time.Time, err error) {
	elapsed := time.Since(start).Milliseconds()
	status := "success"
	if err != nil {
		status = "error"
	}
	RedisJSONSetLatency.WithLabelValues(status).Observe(float64(elapsed))
}

// RecordJSONGetLatency records the latency of a JSON.GET operation.
func RecordJSONGetLatency(start time.Time, err error) {
	elapsed := time.Since(start).Milliseconds()
	status := "success"
	if err != nil {
		status = "error"
	}
	RedisJSONGetLatency.WithLabelValues(status).Observe(float64(elapsed))
}

// RecordCacheHit increments the cache hit counter.
func RecordCacheHit() {
	CacheHits.Inc()
}

// RecordCacheMiss increments the cache miss counter.
func RecordCacheMiss() {
	CacheMisses.Inc()
}

// RecordStreamProcessing records event processing latency and status.
func RecordStreamProcessing(start time.Time, err error, action string) {
	elapsed := time.Since(start).Milliseconds()
	status := "success"
	if err != nil {
		status = "error"
	}
	StreamProcessingLatency.WithLabelValues(status).Observe(float64(elapsed))
	StreamEventsProcessed.WithLabelValues(action).Inc()
}

// RecordBloomCheck records a Bloom filter existence check result.
func RecordBloomCheck(exists bool) {
	result := "not_exists"
	if exists {
		result = "exists"
	}
	BloomFilterChecks.WithLabelValues(result).Inc()
}

// RecordSearchLatency records full-text search latency.
func RecordSearchLatency(start time.Time, err error) {
	elapsed := time.Since(start).Milliseconds()
	status := "success"
	if err != nil {
		status = "error"
	}
	RediSearchLatency.WithLabelValues(status).Observe(float64(elapsed))
}
