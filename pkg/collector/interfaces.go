package collector

import "context"

type CIMetricsCollector interface {
	Collect(cancel context.CancelFunc) error
}
