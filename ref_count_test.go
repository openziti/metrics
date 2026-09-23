package metrics

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_RefCount(t *testing.T) {
	reg := NewRegistry("test", nil)
	require.False(t, reg.IsValidMetric("test"))

	th := reg.RefCountedHistogram("test")
	require.True(t, reg.IsValidMetric("test"))

	th.Dispose()
	require.False(t, reg.IsValidMetric("test"))

	th = reg.RefCountedHistogram("test")
	require.True(t, reg.IsValidMetric("test"))

	th.Dispose()
	require.False(t, reg.IsValidMetric("test"))

	th = reg.RefCountedHistogram("test")
	th2 := reg.RefCountedHistogram("test")
	require.True(t, reg.IsValidMetric("test"))

	th.Dispose()
	require.True(t, reg.IsValidMetric("test"))

	th2.Dispose()
	require.False(t, reg.IsValidMetric("test"))
}

func Test_RefCountedMeterSurvivesOwnerReplacement(t *testing.T) {
	reg := NewRegistry("test", nil)

	outgoing := reg.RefCountedMeter("link.1")
	outgoing.Mark(1)

	replacement := reg.RefCountedMeter("link.1")
	require.Same(t, outgoing, replacement)

	outgoing.Dispose()
	require.True(t, reg.IsValidMetric("link.1"))
	replacement.Mark(1)
	require.Equal(t, int64(2), replacement.Count())

	replacement.Dispose()
	require.False(t, reg.IsValidMetric("link.1"))
	replacement.Mark(1)
	require.Equal(t, int64(2), replacement.Count(), "meter should be stopped once the last reference is released")
}

func Test_MeterLookupTakesNoReference(t *testing.T) {
	reg := NewRegistry("test", nil)

	m := reg.Meter("events")
	for i := 0; i < 10; i++ {
		reg.Meter("events").Mark(1)
	}
	require.Equal(t, int64(10), m.Count())

	m.Dispose()
	require.False(t, reg.IsValidMetric("events"))
	m.Mark(1)
	require.Equal(t, int64(10), m.Count(), "meter should be stopped on dispose")
}

func Test_HistogramLookupTakesNoReference(t *testing.T) {
	reg := NewRegistry("test", nil)

	h := reg.Histogram("sizes")
	for i := 0; i < 10; i++ {
		reg.Histogram("sizes").Update(int64(i))
	}
	require.Equal(t, int64(10), h.Count())

	h.Dispose()
	require.False(t, reg.IsValidMetric("sizes"))
}

func Test_DisposeAllTearsDownRefCounted(t *testing.T) {
	reg := NewRegistry("test", nil)

	m := reg.RefCountedMeter("m")
	reg.RefCountedMeter("m")
	plain := reg.Meter("plain")

	reg.DisposeAll()
	require.False(t, reg.IsValidMetric("m"))
	require.False(t, reg.IsValidMetric("plain"))

	m.Mark(1)
	require.Equal(t, int64(0), m.Count(), "meter with outstanding references is stopped by DisposeAll")
	plain.Mark(1)
	require.Equal(t, int64(0), plain.Count())

	// a stale handle's Dispose must not disturb a replacement created under the same name
	replacement := reg.RefCountedMeter("m")
	m.Dispose()
	require.True(t, reg.IsValidMetric("m"))
	replacement.Mark(1)
	require.Equal(t, int64(1), replacement.Count())
}

func Test_MixedAccessorKindsPanic(t *testing.T) {
	reg := NewRegistry("test", nil)

	reg.Meter("m")
	require.Panics(t, func() { reg.RefCountedMeter("m") })
	require.Panics(t, func() { reg.Histogram("m") })
	require.Panics(t, func() { reg.RefCountedHistogram("m") })

	reg.RefCountedMeter("rm")
	require.Panics(t, func() { reg.Meter("rm") })
	require.Panics(t, func() { reg.RefCountedHistogram("rm") })
	require.Panics(t, func() { reg.Timer("rm") })

	reg.RefCountedHistogram("rh")
	require.Panics(t, func() { reg.Histogram("rh") })
	require.Panics(t, func() { reg.RefCountedMeter("rh") })

	// a failed mismatched lookup must not have taken a reference
	rh := reg.GetHistogram("rh")
	rh.Dispose()
	require.False(t, reg.IsValidMetric("rh"))
}

func Test_VisitorSeesBothMeterKinds(t *testing.T) {
	reg := NewRegistry("test", nil)
	reg.Meter("m").Mark(1)
	reg.RefCountedMeter("rm").Mark(2)
	reg.Histogram("h").Update(1)
	reg.RefCountedHistogram("rh").Update(2)

	visitor := newCollectingVisitor()
	reg.AcceptVisitor(visitor)
	require.Equal(t, int64(1), visitor.meters["m"].Count())
	require.Equal(t, int64(2), visitor.meters["rm"].Count())
	require.Equal(t, int64(1), visitor.histograms["h"].Sum())
	require.Equal(t, int64(2), visitor.histograms["rh"].Sum())
}
