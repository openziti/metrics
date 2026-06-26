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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// collectingVisitor records which metrics AcceptVisitor walks, so the tests can
// assert the collection-only registry exposes each metric type to a Visitor.
type collectingVisitor struct {
	gauges     map[string]Gauge
	floatGauge map[string]GaugeFloat64
	meters     map[string]Meter
	histograms map[string]Histogram
	timers     map[string]Timer
}

func newCollectingVisitor() *collectingVisitor {
	return &collectingVisitor{
		gauges:     map[string]Gauge{},
		floatGauge: map[string]GaugeFloat64{},
		meters:     map[string]Meter{},
		histograms: map[string]Histogram{},
		timers:     map[string]Timer{},
	}
}

func (v *collectingVisitor) VisitGauge(name string, gauge Gauge)           { v.gauges[name] = gauge }
func (v *collectingVisitor) VisitGaugeFloat64(name string, g GaugeFloat64) { v.floatGauge[name] = g }
func (v *collectingVisitor) VisitMeter(name string, meter Meter)           { v.meters[name] = meter }
func (v *collectingVisitor) VisitHistogram(name string, histogram Histogram) {
	v.histograms[name] = histogram
}
func (v *collectingVisitor) VisitTimer(name string, timer Timer) { v.timers[name] = timer }

func TestAcceptVisitorEmpty(t *testing.T) {
	registry := NewRegistry("test", nil)
	visitor := newCollectingVisitor()
	registry.AcceptVisitor(visitor)
	require.Empty(t, visitor.gauges)
	require.Empty(t, visitor.meters)
	require.Empty(t, visitor.histograms)
	require.Empty(t, visitor.timers)
}

func TestAcceptVisitorWalksAllTypes(t *testing.T) {
	registry := NewRegistry("test", nil)

	registry.Gauge("gauge").Update(3)
	registry.GaugeFloat64("floatGauge").Update(1.5)
	registry.Meter("meter").Mark(1)
	registry.Histogram("histogram").Update(10)
	registry.Timer("timer").Update(time.Second)

	visitor := newCollectingVisitor()
	registry.AcceptVisitor(visitor)

	require.Contains(t, visitor.gauges, "gauge")
	require.Equal(t, int64(3), visitor.gauges["gauge"].Value())
	require.Contains(t, visitor.floatGauge, "floatGauge")
	require.Contains(t, visitor.meters, "meter")
	require.Contains(t, visitor.histograms, "histogram")
	require.Contains(t, visitor.timers, "timer")
}
