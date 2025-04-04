package metrics

import (
	"github.com/rcrowley/go-metrics"
)

// Gauge represents a metric which is measuring a count and a rate
type Gauge interface {
	Metric
	Value() int64
	Update(int64)
}

type gaugeImpl struct {
	metrics.Gauge
	dispose func()
}

func (gauge *gaugeImpl) Dispose() {
	gauge.dispose()
}
