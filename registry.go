/*
	Copyright NetFoundry Inc.

	Licensed under the Apache License, Version 2.0 (the "License");
	you may not use this file except in compliance with the License.
	You may obtain a copy of the License at

	https://www.apache.org/licenses/LICENSE-2.0

	Unless required by applicable law or agreed to in writing, software
	distributed under the License is distributed on an "AS IS" BASIS,
	WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
	See the License for the specific language governing permissions and
	limitations under the License.
*/

package metrics

import (
	"fmt"
	"reflect"

	"github.com/openziti/foundation/v2/logging"
	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/rcrowley/go-metrics"
)

// log is the package-level logger for the metrics package.
var log = logging.For("metrics")

// Metric is the base functionality for all metrics types
type Metric interface {
	// Dispose removes this metric from its Registry and releases any resources it holds. A metric obtained
	// from RefCountedMeter or RefCountedHistogram is instead released by one reference and torn down only
	// once the last is gone.
	Dispose()
}

// Registry allows for configuring and accessing metrics for an application.
//
// Every accessor returns the existing metric for a name or creates one, and may be called as often as
// is convenient, except RefCountedMeter and RefCountedHistogram. Those take a reference on every call,
// including calls that find the metric already present, and Dispose releases one. Each owner of a
// reference counted metric resolves it once, holds it, and disposes it once.
//
// Reference counting is for metrics whose owner can be replaced under the same name, where the
// replacement may resolve the metric before the outgoing owner has disposed it. The outgoing Dispose
// releases only its own reference, so the metric survives for the replacement rather than being torn
// down beneath it. A metric with a single, stable owner gains nothing from it.
//
// A name is bound to the kind of accessor that created it. Resolving it through an accessor of a
// different kind, including Meter against a name created by RefCountedMeter or the reverse, panics.
type Registry interface {
	// SourceId returns the source id of this Registry
	SourceId() string

	// Gauge returns a Gauge for the given name. If one does not yet exist, one will be created
	Gauge(name string) Gauge

	// FuncGauge returns a Gauge for the given name. If one does not yet exist, one will be created using
	// the given function
	FuncGauge(name string, f func() int64) Gauge

	// GaugeFloat64 returns a GaugeFloat64 for the given name. If one does not yet exist, one will be created
	GaugeFloat64(name string) GaugeFloat64

	// FuncGaugeFloat64 returns a GaugeFloat64 for the given name. If one does not yet exist, one will be created
	// using the given function
	FuncGaugeFloat64(name string, f func() float64) GaugeFloat64

	// Meter returns a Meter for the given name, creating one if it does not yet exist. Dispose removes it.
	Meter(name string) Meter

	// RefCountedMeter returns a Meter for the given name, creating one if it does not yet exist, and takes
	// a reference on it. Dispose releases one reference; the Meter is removed and stopped once the last is
	// released. Call this once per owner and hold the result, since a call on a per-event path accumulates
	// references that are never released and the Meter then outlives every owner.
	RefCountedMeter(name string) Meter

	// Histogram returns a Histogram for the given name, creating one if it does not yet exist. Dispose
	// removes it.
	Histogram(name string) Histogram

	// RefCountedHistogram returns a Histogram for the given name, creating one if it does not yet exist,
	// and takes a reference on it. Dispose releases one reference; the Histogram is removed once the last
	// is released. Call this once per owner and hold the result, since a call on a per-event path
	// accumulates references that are never released and the Histogram then outlives every owner.
	RefCountedHistogram(name string) Histogram

	// Timer returns a Timer for the given name, creating one if it does not yet exist. Dispose removes it.
	Timer(name string) Timer

	// EachMetric calls the given visitor function for each Metric in this registry
	EachMetric(visitor func(name string, metric Metric))

	// GetGauge returns the Gauge for the given name or nil if a Gauge with that name doesn't exist
	GetGauge(name string) Gauge

	// GetGaugeFloat64 returns the GaugeFloat64 for the given name or nil if one doesn't exist
	GetGaugeFloat64(name string) GaugeFloat64

	// GetMeter returns the Meter for the given name or nil if a Meter with that name doesn't exist
	GetMeter(name string) Meter

	// GetHistogram returns the Histogram for the given name or nil if a Histogram with that name doesn't exist
	GetHistogram(name string) Histogram

	// GetTimer returns the Timer for the given name or nil if a Timer with that name doesn't exist
	GetTimer(name string) Timer

	// IsValidMetric returns true if a metric with the given name exists in the registry, false otherwise
	IsValidMetric(name string) bool

	AcceptVisitor(visitor Visitor)

	// DisposeAll tears down every metric and clears the Registry. Reference counted metrics are torn down
	// regardless of outstanding references; handles still held afterwards refer to stopped metrics, and
	// disposing them is harmless.
	DisposeAll()
}

type Visitor interface {
	VisitGauge(name string, gauge Gauge)
	VisitGaugeFloat64(name string, gauge GaugeFloat64)
	VisitMeter(name string, meter Meter)
	VisitHistogram(name string, histogram Histogram)
	VisitTimer(name string, timer Timer)
}

func NewRegistry(sourceId string, tags map[string]string) Registry {
	return &registryImpl{
		sourceId:  sourceId,
		tags:      tags,
		metricMap: cmap.New[Metric](),
	}
}

type registryImpl struct {
	sourceId  string
	tags      map[string]string
	metricMap cmap.ConcurrentMap[string, Metric]
}

func (registry *registryImpl) dispose(name string) {
	registry.metricMap.Remove(name)
}

func (registry *registryImpl) DisposeAll() {
	items := registry.metricMap.Items()
	registry.metricMap.Clear()
	for _, metric := range items {
		if rc, ok := metric.(refCounted); ok {
			rc.stop()
		} else {
			metric.Dispose()
		}
	}
}

func (registry *registryImpl) IsValidMetric(name string) bool {
	return registry.metricMap.Has(name)
}

func (registry *registryImpl) SourceId() string {
	return registry.sourceId
}

func (registry *registryImpl) GetGauge(name string) Gauge {
	metric, found := registry.metricMap.Get(name)
	if !found {
		return nil
	}
	if gauge, ok := metric.(Gauge); ok {
		return gauge
	}
	return nil
}

func (registry *registryImpl) GetGaugeFloat64(name string) GaugeFloat64 {
	metric, found := registry.metricMap.Get(name)
	if !found {
		return nil
	}
	if gauge, ok := metric.(GaugeFloat64); ok {
		return gauge
	}
	return nil
}

func (registry *registryImpl) GetMeter(name string) Meter {
	metric, found := registry.metricMap.Get(name)
	if !found {
		return nil
	}
	if meter, ok := metric.(Meter); ok {
		return meter
	}
	return nil
}

func (registry *registryImpl) GetHistogram(name string) Histogram {
	metric, found := registry.metricMap.Get(name)
	if !found {
		return nil
	}
	if histogram, ok := metric.(Histogram); ok {
		return histogram
	}
	return nil
}

func (registry *registryImpl) GetTimer(name string) Timer {
	metric, found := registry.metricMap.Get(name)
	if !found {
		return nil
	}
	if timer, ok := metric.(Timer); ok {
		return timer
	}
	return nil
}

func getOrCreateMetric[T Metric](registry *registryImpl, name string, newMetric func() T) T {
	var result T
	for {
		metric, present := registry.metricMap.Get(name)
		if present {
			var ok bool
			result, ok = metric.(T)
			if !ok {
				panic(fmt.Errorf("metric '%v' already exists and is not a %v. It is a %T", name, reflect.TypeFor[T](), metric))
			}
			return result
		}

		result = newMetric()
		if registry.metricMap.SetIfAbsent(name, result) {
			return result
		}
	}
}

func (registry *registryImpl) Gauge(name string) Gauge {
	return getOrCreateMetric(registry, name, func() Gauge {
		return &gaugeImpl{
			Gauge: metrics.NewGauge(),
			dispose: func() {
				registry.dispose(name)
			},
		}
	})
}

func (registry *registryImpl) FuncGauge(name string, f func() int64) Gauge {
	return getOrCreateMetric(registry, name, func() Gauge {
		return &gaugeImpl{
			Gauge: metrics.NewFunctionalGauge(f),
			dispose: func() {
				registry.dispose(name)
			},
		}
	})
}

func (registry *registryImpl) GaugeFloat64(name string) GaugeFloat64 {
	return getOrCreateMetric(registry, name, func() GaugeFloat64 {
		return &gaugeFloat64Impl{
			GaugeFloat64: metrics.NewGaugeFloat64(),
			dispose: func() {
				registry.dispose(name)
			},
		}
	})
}

func (registry *registryImpl) FuncGaugeFloat64(name string, f func() float64) GaugeFloat64 {
	return getOrCreateMetric(registry, name, func() GaugeFloat64 {
		return &gaugeFloat64Impl{
			GaugeFloat64: metrics.NewFunctionalGaugeFloat64(f),
			dispose: func() {
				registry.dispose(name)
			},
		}
	})
}

func (registry *registryImpl) Meter(name string) Meter {
	return getOrCreateMetric(registry, name, func() *meterImpl {
		return &meterImpl{
			Meter: metrics.NewMeter(),
			dispose: func() {
				registry.dispose(name)
			},
		}
	})
}

func (registry *registryImpl) RefCountedMeter(name string) Meter {
	return getOrCreateRefCounted(registry, name, func() *refCountedMeterImpl {
		return &refCountedMeterImpl{
			Meter:    metrics.NewMeter(),
			registry: registry,
			name:     name,
		}
	})
}

func newHistogram() metrics.Histogram {
	return metrics.NewHistogram(metrics.NewExpDecaySample(128, 0.015))
}

func (registry *registryImpl) Histogram(name string) Histogram {
	return getOrCreateMetric(registry, name, func() *histogramImpl {
		return &histogramImpl{
			Histogram: newHistogram(),
			dispose: func() {
				registry.dispose(name)
			},
		}
	})
}

func (registry *registryImpl) RefCountedHistogram(name string) Histogram {
	return getOrCreateRefCounted(registry, name, func() *refCountedHistogramImpl {
		return &refCountedHistogramImpl{
			Histogram: newHistogram(),
			registry:  registry,
			name:      name,
		}
	})
}

// getOrCreateRefCounted resolves the metric under name, creating it with factory if absent, and takes a
// reference on it. Panics if the name holds a metric that is not a T; the check runs before the
// reference is taken, and outside the map callback since the shard lock is not released on panic.
func getOrCreateRefCounted[T refCounted](registry *registryImpl, name string, factory func() T) T {
	var mismatch Metric
	metric := registry.metricMap.Upsert(name, nil, func(exist bool, valueInMap Metric, newValue Metric) Metric {
		if exist {
			if v, ok := valueInMap.(T); ok {
				v.IncrRefCount()
			} else {
				mismatch = valueInMap
			}
			return valueInMap
		}

		newVal := factory()
		newVal.IncrRefCount()
		return newVal
	})

	if mismatch != nil {
		panic(fmt.Errorf("metric '%v' already exists and is not a %v. It is a %T", name, reflect.TypeFor[T](), mismatch))
	}
	return metric.(T)
}

func (registry *registryImpl) disposeRefCounted(metric refCounted) {
	removed := registry.metricMap.RemoveCb(metric.Name(), func(key string, v Metric, exists bool) bool {
		if !exists {
			return true
		}
		return v == metric && metric.DecrRefCount() < 1
	})

	if removed {
		metric.stop()
	}
}

func (registry *registryImpl) Timer(name string) Timer {
	return getOrCreateMetric(registry, name, func() Timer {
		return &timerImpl{
			Timer: metrics.NewTimer(),
			dispose: func() {
				registry.dispose(name)
			},
		}
	})
}

func (registry *registryImpl) EachMetric(visitor func(name string, metric Metric)) {
	for entry := range registry.metricMap.IterBuffered() {
		visitor(entry.Key, entry.Val)
	}
}

func (registry *registryImpl) Each(visitor func(string, interface{})) {
	for entry := range registry.metricMap.IterBuffered() {
		visitor(entry.Key, entry.Val)
	}
}

// Provide rest of go-metrics Registry interface, so we can use go-metrics reporters if desired
func (registry *registryImpl) Get(s string) interface{} {
	val, _ := registry.metricMap.Get(s)
	return unwrapMetric(val)
}

func (registry *registryImpl) GetAll() map[string]map[string]interface{} {
	return nil
}

func (registry *registryImpl) GetOrRegister(s string, i interface{}) interface{} {
	return registry.metricMap.Upsert(s, wrapMetric(i), func(exist bool, valueInMap Metric, newValue Metric) Metric {
		if exist {
			return valueInMap
		}
		return newValue
	})
}

func (registry *registryImpl) Register(s string, i interface{}) error {
	if registry.metricMap.SetIfAbsent(s, wrapMetric(i)) {
		return fmt.Errorf("duplicate metric %v", s)
	}
	return nil
}

func (registry *registryImpl) RunHealthchecks() {
}

func (registry *registryImpl) Unregister(s string) {
	registry.metricMap.Remove(s)
}

func (registry *registryImpl) UnregisterAll() {
	for _, key := range registry.metricMap.Keys() {
		registry.Unregister(key)
	}
}

func (registry *registryImpl) AcceptVisitor(visitor Visitor) {
	// If there's nothing to report, skip it
	if registry.metricMap.Count() == 0 {
		return
	}

	registry.EachMetric(func(name string, i Metric) {
		switch metric := i.(type) {
		case *gaugeImpl:
			visitor.VisitGauge(name, metric)
		case *gaugeFloat64Impl:
			visitor.VisitGaugeFloat64(name, metric)
		case *meterImpl:
			visitor.VisitMeter(name, metric)
		case *refCountedMeterImpl:
			visitor.VisitMeter(name, metric)
		case *histogramImpl:
			visitor.VisitHistogram(name, metric.CreateSnapshot())
		case *refCountedHistogramImpl:
			visitor.VisitHistogram(name, metric.CreateSnapshot())
		case *timerImpl:
			visitor.VisitTimer(name, metric.CreateSnapshot())
		default:
			log.Error("unsupported metric type", "type", reflect.TypeOf(i))
		}
	})
}

type refCounted interface {
	Metric
	IncrRefCount() int32
	DecrRefCount() int32
	Name() string
	stop()
}

func wrapMetric(v any) Metric {
	if v == nil {
		return nil
	}
	if m, ok := v.(Metric); ok {
		return m
	}
	return metricWrapper{
		value: v,
	}
}

func unwrapMetric(m Metric) any {
	if v, ok := m.(metricWrapper); ok {
		return v.value
	}
	return m
}

type metricWrapper struct {
	value any
}

func (m metricWrapper) Dispose() {}
