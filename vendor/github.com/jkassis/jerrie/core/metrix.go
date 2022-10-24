package core

import "github.com/prometheus/client_golang/prometheus"

// PromRegisterCollector adds a collector and gracefully handles errors
func PromRegisterCollector(c prometheus.Collector) prometheus.Collector {
	if err := prometheus.Register(c); err != nil {
		switch err.(type) {
		case prometheus.AlreadyRegisteredError:
			return err.(prometheus.AlreadyRegisteredError).ExistingCollector
		default:
			panic(err)
		}
	}
	return c
}
