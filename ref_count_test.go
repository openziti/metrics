package metrics

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_RefCount(t *testing.T) {
	reg := NewRegistry("test", nil)
	require.False(t, reg.IsValidMetric("test"))

	th := reg.Histogram("test")
	require.True(t, reg.IsValidMetric("test"))

	th.Dispose()
	require.False(t, reg.IsValidMetric("test"))

	th = reg.Histogram("test")
	require.True(t, reg.IsValidMetric("test"))

	th.Dispose()
	require.False(t, reg.IsValidMetric("test"))

	th = reg.Histogram("test")
	th2 := reg.Histogram("test")
	require.True(t, reg.IsValidMetric("test"))

	th.Dispose()
	require.True(t, reg.IsValidMetric("test"))

	th2.Dispose()
	require.False(t, reg.IsValidMetric("test"))
}
