# Ziti Metrics Library

This is a metrics library which is built on, and extends the
[go-metrics](https://github.com/rcrowley/go-metrics) library.

It extends it by adding the following:

1. A Dispose method is defined on metrics, so they can be cleaned up, for metrics tied to transient entities.
1. Reference counted metrics
1. A Visitor over the registry, so collected metrics can be reported to any sink.

## v2

`v2` is collection-only. The metrics wire format (the `MetricsMessage` protobuf,
the message builder, and the interval/usage counter reporting subsystem) has been
removed; consumers that need to serialize metrics own that format themselves and
read a registry through `AcceptVisitor`. Import as
`github.com/openziti/metrics/v2`.

## v3

`v3` separates reference counted metrics from plain ones. `Meter` and `Histogram` behave like
every other accessor: they return the existing metric or create it, may be called as often as
is convenient, and `Dispose` removes the metric. `RefCountedMeter` and `RefCountedHistogram`
take a reference on every call and `Dispose` releases one; the metric is torn down when the last
reference goes. Use those for metrics whose owner can be replaced under the same name, so a
replacement that has already resolved the metric does not have it torn down by the outgoing
owner's `Dispose`. A name is bound to the accessor kind that created it, and resolving it through
the other kind panics. Import as `github.com/openziti/metrics/v3`.
