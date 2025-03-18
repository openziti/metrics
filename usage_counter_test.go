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
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type intervalWaiter struct {
	duration        time.Duration
	startInterval   time.Time
	currentInterval time.Time
}

func (self *intervalWaiter) waitForFirstInterval() {
	now := time.Now()
	interval := now.Truncate(self.duration)
	if now == interval {
		self.currentInterval = now
	} else {
		self.currentInterval = interval.Add(self.duration)
		time.Sleep(time.Until(self.currentInterval))
	}
	self.startInterval = self.currentInterval
}

func (self *intervalWaiter) waitForNextInterval() {
	self.currentInterval = self.currentInterval.Add(self.duration)
	if waitTime := time.Until(self.currentInterval); waitTime > 0 {
		time.Sleep(waitTime)
	}
}

func newTestSource(intervalId string, tags map[string]string) UsageSource {
	return &testSource{
		intervalId: intervalId,
		tags:       tags,
	}
}

type testSource struct {
	intervalId string
	tags       map[string]string
}

func (t testSource) GetIntervalId() string {
	return t.intervalId
}

func (t testSource) GetTags() map[string]string {
	return t.tags
}

func TestUsageCounterSingleInterval(t *testing.T) {
	req := require.New(t)

	closeNotify := make(chan struct{})
	defer close(closeNotify)

	interval := time.Second

	config := DefaultUsageRegistryConfig("test", closeNotify)
	registry := NewUsageRegistry(config).(*usageRegistryImpl)
	registry.StartReporting(nil, time.Hour, 10)
	usageCounter := registry.UsageCounter("usage", interval)
	waiter := &intervalWaiter{duration: interval}
	waiter.waitForFirstInterval()

	usageSource1 := newTestSource(uuid.NewString(), map[string]string{
		"serviceId":       uuid.NewString(),
		"identity":        uuid.NewString(),
		"hostingIdentity": uuid.NewString(),
	})

	usageSource2 := newTestSource(uuid.NewString(), map[string]string{
		"serviceId": uuid.NewString(),
	})

	// interval 1
	usageCounter.Update(usageSource1, "ingress.rx", time.Now(), 100)
	usageCounter.Update(usageSource1, "egress.tx", time.Now(), 100)
	usageCounter.Update(usageSource2, "ingress.rx", time.Now(), 120)
	usageCounter.Update(usageSource2, "egress.tx", time.Now(), 120)

	time.Sleep(100 * time.Millisecond)

	usageCounter.Update(usageSource1, "ingress.rx", time.Now(), 200)
	usageCounter.Update(usageSource1, "egress.tx", time.Now(), 200)
	usageCounter.Update(usageSource2, "ingress.rx", time.Now(), 80)
	usageCounter.Update(usageSource2, "egress.tx", time.Now(), 80)

	time.Sleep(200 * time.Millisecond)
	usageCounter.Update(usageSource1, "ingress.rx", time.Now(), 50)
	usageCounter.Update(usageSource1, "egress.tx", time.Now(), 55)

	time.Sleep(400 * time.Millisecond)
	usageCounter.Update(usageSource2, "ingress.rx", time.Now(), 180)
	usageCounter.Update(usageSource2, "egress.tx", time.Now(), 180)

	waiter.waitForNextInterval()
	msg := registry.FlushAndPoll()
	req.Equal(1, len(msg.UsageCounters))

	counter := msg.UsageCounters[0]
	req.Equal(waiter.startInterval.Unix(), counter.IntervalStartUTC)
	req.Equal(uint64(waiter.duration.Seconds()), counter.IntervalLength)
	req.Equal(2, len(counter.Buckets))
	req.NotNil(counter.Buckets[usageSource1.GetIntervalId()])
	bucket1 := counter.Buckets[usageSource1.GetIntervalId()]
	req.Equal(uint64(350), bucket1.Values["ingress.rx"])
	req.Equal(uint64(355), bucket1.Values["egress.tx"])
	req.NotNil(counter.Buckets[usageSource2.GetIntervalId()])
	bucket2 := counter.Buckets[usageSource2.GetIntervalId()]
	req.Equal(uint64(380), bucket2.Values["ingress.rx"])
	req.Equal(uint64(380), bucket2.Values["egress.tx"])
}

func TestUsageCounterTwoIntervals(t *testing.T) {
	req := require.New(t)

	closeNotify := make(chan struct{})
	defer close(closeNotify)

	interval := time.Second

	config := DefaultUsageRegistryConfig("test", closeNotify)
	registry := NewUsageRegistry(config).(*usageRegistryImpl)
	registry.StartReporting(nil, time.Hour, 10)
	usageCounter := registry.UsageCounter("usage", interval)
	waiter := &intervalWaiter{duration: interval}
	waiter.waitForFirstInterval()

	usageSource1 := newTestSource(uuid.NewString(), map[string]string{
		"serviceId":       uuid.NewString(),
		"identity":        uuid.NewString(),
		"hostingIdentity": uuid.NewString(),
	})

	usageSource2 := newTestSource(uuid.NewString(), map[string]string{
		"serviceId": uuid.NewString(),
	})

	// interval 1
	usageCounter.Update(usageSource1, "ingress.rx", time.Now(), 100)
	usageCounter.Update(usageSource1, "egress.tx", time.Now(), 100)
	usageCounter.Update(usageSource2, "ingress.rx", time.Now(), 120)
	usageCounter.Update(usageSource2, "egress.tx", time.Now(), 120)

	time.Sleep(100 * time.Millisecond)

	usageCounter.Update(usageSource1, "ingress.rx", time.Now(), 200)
	usageCounter.Update(usageSource1, "egress.tx", time.Now(), 200)
	usageCounter.Update(usageSource2, "ingress.rx", time.Now(), 80)
	usageCounter.Update(usageSource2, "egress.tx", time.Now(), 80)

	time.Sleep(100 * time.Millisecond)
	usageCounter.Update(usageSource1, "ingress.rx", time.Now(), 50)
	usageCounter.Update(usageSource1, "egress.tx", time.Now(), 55)

	time.Sleep(400 * time.Millisecond)
	usageCounter.Update(usageSource2, "ingress.rx", time.Now(), 180)
	usageCounter.Update(usageSource2, "egress.tx", time.Now(), 180)

	waiter.waitForNextInterval()
	msg := registry.FlushAndPoll()
	req.Equal(1, len(msg.UsageCounters))

	counter := msg.UsageCounters[0]
	req.Equal(waiter.startInterval.Unix(), counter.IntervalStartUTC)
	req.Equal(uint64(waiter.duration.Seconds()), counter.IntervalLength)
	req.Equal(2, len(counter.Buckets))
	req.NotNil(counter.Buckets[usageSource1.GetIntervalId()])
	bucket1 := counter.Buckets[usageSource1.GetIntervalId()]
	req.Equal(uint64(350), bucket1.Values["ingress.rx"])
	req.Equal(uint64(355), bucket1.Values["egress.tx"])
	req.NotNil(counter.Buckets[usageSource2.GetIntervalId()])
	bucket2 := counter.Buckets[usageSource2.GetIntervalId()]
	req.Equal(uint64(380), bucket2.Values["ingress.rx"])
	req.Equal(uint64(380), bucket2.Values["egress.tx"])

	// interval 2
	usageCounter.Update(usageSource1, "ingress.rx", time.Now(), 200)
	usageCounter.Update(usageSource1, "egress.tx", time.Now(), 200)
	usageCounter.Update(usageSource2, "ingress.rx", time.Now(), 220)
	usageCounter.Update(usageSource2, "egress.tx", time.Now(), 220)

	time.Sleep(200 * time.Millisecond)

	usageCounter.Update(usageSource1, "ingress.rx", time.Now(), 200)
	usageCounter.Update(usageSource1, "egress.tx", time.Now(), 200)
	usageCounter.Update(usageSource2, "ingress.rx", time.Now(), 80)
	usageCounter.Update(usageSource2, "egress.tx", time.Now(), 80)

	time.Sleep(100 * time.Millisecond)
	usageCounter.Update(usageSource1, "ingress.rx", time.Now(), 50)
	usageCounter.Update(usageSource1, "egress.tx", time.Now(), 55)

	time.Sleep(300 * time.Millisecond)
	usageCounter.Update(usageSource2, "ingress.rx", time.Now(), 180)
	usageCounter.Update(usageSource2, "egress.tx", time.Now(), 180)

	// We shouldn't get additional intervals
	waiter.waitForNextInterval()
	msg = registry.FlushAndPoll()
	req.Equal(1, len(msg.UsageCounters))

	counter = msg.UsageCounters[0]
	req.Equal(waiter.startInterval.Add(interval).Unix(), counter.IntervalStartUTC)
	req.Equal(uint64(waiter.duration.Seconds()), counter.IntervalLength)
	req.Equal(2, len(counter.Buckets))
	req.NotNil(counter.Buckets[usageSource1.GetIntervalId()])
	bucket1 = counter.Buckets[usageSource1.GetIntervalId()]
	req.Equal(uint64(450), bucket1.Values["ingress.rx"])
	req.Equal(uint64(455), bucket1.Values["egress.tx"])
	req.NotNil(counter.Buckets[usageSource2.GetIntervalId()])
	bucket2 = counter.Buckets[usageSource2.GetIntervalId()]
	req.Equal(uint64(480), bucket2.Values["ingress.rx"])
	req.Equal(uint64(480), bucket2.Values["egress.tx"])
}

func TestUsageCounterTwoIntervalsSinglePoll(t *testing.T) {
	req := require.New(t)

	closeNotify := make(chan struct{})
	defer close(closeNotify)

	interval := time.Second

	config := DefaultUsageRegistryConfig("test", closeNotify)
	registry := NewUsageRegistry(config).(*usageRegistryImpl)
	registry.StartReporting(nil, time.Hour, 10)
	usageCounter := registry.UsageCounter("usage", interval)
	waiter := &intervalWaiter{duration: interval}
	waiter.waitForFirstInterval()

	usageSource1 := newTestSource(uuid.NewString(), map[string]string{
		"serviceId":       uuid.NewString(),
		"identity":        uuid.NewString(),
		"hostingIdentity": uuid.NewString(),
	})

	usageSource2 := newTestSource(uuid.NewString(), map[string]string{
		"serviceId": uuid.NewString(),
	})

	// interval 1
	usageCounter.Update(usageSource1, "ingress.rx", time.Now(), 100)
	usageCounter.Update(usageSource1, "egress.tx", time.Now(), 100)
	usageCounter.Update(usageSource2, "ingress.rx", time.Now(), 120)
	usageCounter.Update(usageSource2, "egress.tx", time.Now(), 120)

	time.Sleep(100 * time.Millisecond)

	usageCounter.Update(usageSource1, "ingress.rx", time.Now(), 200)
	usageCounter.Update(usageSource1, "egress.tx", time.Now(), 200)
	usageCounter.Update(usageSource2, "ingress.rx", time.Now(), 80)
	usageCounter.Update(usageSource2, "egress.tx", time.Now(), 80)

	time.Sleep(200 * time.Millisecond)
	usageCounter.Update(usageSource1, "ingress.rx", time.Now(), 50)
	usageCounter.Update(usageSource1, "egress.tx", time.Now(), 55)

	time.Sleep(400 * time.Millisecond)
	usageCounter.Update(usageSource2, "ingress.rx", time.Now(), 180)
	usageCounter.Update(usageSource2, "egress.tx", time.Now(), 180)

	waiter.waitForNextInterval()

	// interval 2
	usageCounter.Update(usageSource1, "ingress.rx", time.Now(), 200)
	usageCounter.Update(usageSource1, "egress.tx", time.Now(), 200)
	usageCounter.Update(usageSource2, "ingress.rx", time.Now(), 220)
	usageCounter.Update(usageSource2, "egress.tx", time.Now(), 220)

	time.Sleep(200 * time.Millisecond)

	usageCounter.Update(usageSource1, "ingress.rx", time.Now(), 200)
	usageCounter.Update(usageSource1, "egress.tx", time.Now(), 200)
	usageCounter.Update(usageSource2, "ingress.rx", time.Now(), 80)
	usageCounter.Update(usageSource2, "egress.tx", time.Now(), 80)

	time.Sleep(100 * time.Millisecond)
	usageCounter.Update(usageSource1, "ingress.rx", time.Now(), 50)
	usageCounter.Update(usageSource1, "egress.tx", time.Now(), 55)

	time.Sleep(300 * time.Millisecond)
	usageCounter.Update(usageSource2, "ingress.rx", time.Now(), 180)
	usageCounter.Update(usageSource2, "egress.tx", time.Now(), 180)

	// We shouldn't get additional intervals
	waiter.waitForNextInterval()
	msg := registry.FlushAndPoll()
	req.Equal(2, len(msg.UsageCounters))

	counter := msg.UsageCounters[0]
	req.Equal(waiter.startInterval.Unix(), counter.IntervalStartUTC)
	req.Equal(uint64(waiter.duration.Seconds()), counter.IntervalLength)
	req.Equal(2, len(counter.Buckets))
	req.NotNil(counter.Buckets[usageSource1.GetIntervalId()])
	bucket1 := counter.Buckets[usageSource1.GetIntervalId()]
	req.Equal(uint64(350), bucket1.Values["ingress.rx"])
	req.Equal(uint64(355), bucket1.Values["egress.tx"])
	req.NotNil(counter.Buckets[usageSource2.GetIntervalId()])
	bucket2 := counter.Buckets[usageSource2.GetIntervalId()]
	req.Equal(uint64(380), bucket2.Values["ingress.rx"])
	req.Equal(uint64(380), bucket2.Values["egress.tx"])

	counter = msg.UsageCounters[1]
	req.Equal(waiter.startInterval.Add(interval).Unix(), counter.IntervalStartUTC)
	req.Equal(uint64(waiter.duration.Seconds()), counter.IntervalLength)
	req.Equal(2, len(counter.Buckets))
	req.NotNil(counter.Buckets[usageSource1.GetIntervalId()])
	bucket1 = counter.Buckets[usageSource1.GetIntervalId()]
	req.Equal(uint64(450), bucket1.Values["ingress.rx"])
	req.Equal(uint64(455), bucket1.Values["egress.tx"])
	req.NotNil(counter.Buckets[usageSource2.GetIntervalId()])
	bucket2 = counter.Buckets[usageSource2.GetIntervalId()]
	req.Equal(uint64(480), bucket2.Values["ingress.rx"])
	req.Equal(uint64(480), bucket2.Values["egress.tx"])
}
