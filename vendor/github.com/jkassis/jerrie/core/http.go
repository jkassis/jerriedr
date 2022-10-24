package core

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// HTTPClient is a wrapper for http.Client that collects prometheus metrics
type HTTPClient struct {
	http.Client
	metrxRequestDuration *prometheus.HistogramVec
}

// Init initializes an HTTPClient
func (c *HTTPClient) Init() {
	c.metrxRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_client_request_duration",
		Help:    "duration of requests initiated by this http client",
		Buckets: prometheus.LinearBuckets(0, 50000000, 40), // 40 buckets at 50ms each starting at 0ms
	}, []string{"url", "status"})

	PromRegisterCollector(c.metrxRequestDuration)
}

// Do wraps the native http.Do and records metrics
func (c *HTTPClient) Do(req *http.Request) (res *http.Response, err error) {
	start := time.Now()
	res, err = c.Client.Do(req)
	duration := time.Since(start)

	if err != nil {
		c.metrxRequestDuration.WithLabelValues(URLPath(req.URL.Path), "err").Observe(float64(duration))
	} else {
		c.metrxRequestDuration.WithLabelValues(URLPath(req.URL.Path), res.Status).Observe(float64(duration))
	}

	return
}
