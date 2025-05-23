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
	"github.com/openziti/foundation/v2/concurrenz"
	"github.com/rcrowley/go-metrics"
)

// Histogram represents a metric which is measuring the distribution of values for some measurement
type Histogram interface {
	Metric
	Count() int64
	Max() int64
	Mean() float64
	Min() int64
	Percentile(float64) float64
	Percentiles([]float64) []float64
	StdDev() float64
	Sum() int64
	Variance() float64

	Clear()
	Update(int64)
	CreateSnapshot() Histogram
}

type histogramImpl struct {
	metrics.Histogram
	name     string
	registry *registryImpl
	concurrenz.RefCount
}

func (self *histogramImpl) Name() string {
	return self.name
}

func (self *histogramImpl) Dispose() {
	self.registry.disposeRefCounted(self)
}

func (self *histogramImpl) stop() {
	// no resources to cleanup
}

func (self *histogramImpl) CreateSnapshot() Histogram {
	return &histogramSnapshot{
		Histogram: self.Snapshot(),
		name:      self.name,
	}
}

type histogramSnapshot struct {
	metrics.Histogram
	name string
}

func (self *histogramSnapshot) Name() string {
	return self.name
}

func (self *histogramSnapshot) Dispose() {}

func (self *histogramSnapshot) CreateSnapshot() Histogram {
	return self
}
