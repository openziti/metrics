package metrics

import (
	"time"

	"github.com/rcrowley/go-metrics"
)

type Timer interface {
	Metric
	Time(func())
	Update(time.Duration)
	UpdateSince(time.Time)
}

type timerImpl struct {
	metrics.Timer
	dispose func()
}

func (t *timerImpl) Dispose() {
	t.Stop()
	t.dispose()
}
