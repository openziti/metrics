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