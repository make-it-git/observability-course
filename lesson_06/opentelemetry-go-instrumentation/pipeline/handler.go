// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package pipeline provides interfaces and types shared by other packages that
// handle processing and export of auto instrumentation generated telemetry.
// Standard types that implement the included interfaces are found in
// sub-packages.
package pipeline

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// Handler handles telemetry generated by instrumentation.
type Handler struct {
	// TraceHandler is used to handle trace telemetry. This may be nil if the
	// Handler does not support trace telemetry.
	TraceHandler TraceHandler
	// MetricHandler is used to handle metric telemetry. This may be nil if the
	// Handler does not support metric telemetry.
	MetricHandler MetricHandler
	// LogHandler is used to handle log telemetry. This may be nil if the
	// Handler does not support log telemetry.
	LogHandler LogHandler

	scope     pcommon.InstrumentationScope
	schemaURL string
}

// WithScope returns a Handler that includes the given scope and schema url in
// each handle operation (Trace, Metric, Log).
func (h Handler) WithScope(scope pcommon.InstrumentationScope, url string) Handler {
	return Handler{
		TraceHandler:  h.TraceHandler,
		MetricHandler: h.MetricHandler,
		LogHandler:    h.LogHandler,
		scope:         scope,
		schemaURL:     url,
	}
}

// Trace handles the spans by passing them to h's TraceHandler along with the
// configured scope and schema URL of h if h's TraceHandler is not nil.
//
// If h's TraceHandler is nil, the passed spans are dropped.
func (h Handler) Trace(spans ptrace.SpanSlice) {
	if h.TraceHandler == nil {
		return
	}
	h.TraceHandler.HandleTrace(h.scope, h.schemaURL, spans)
}

// Metric handles the metrics by passing them to h's MetricHandler along with
// the configured scope and schema URL of h if h's MetricHandler is not nil.
//
// If h's MetricHandler is nil, the passed metrics are dropped.
func (h Handler) Metric(metrics pmetric.MetricSlice) {
	if h.MetricHandler == nil {
		return
	}
	h.MetricHandler.HandleMetric(h.scope, h.schemaURL, metrics)
}

// Log handles the logs by passing them to h's LogHandler along with the
// configured scope and schema URL of h if h's LogHandler is not nil.
//
// If h's LogHandler is nil, the passed logs are dropped.
func (h Handler) Log(logs plog.LogRecordSlice) {
	if h.LogHandler == nil {
		return
	}
	h.LogHandler.HandleLog(h.scope, h.schemaURL, logs)
}

// TraceHandler handles trace telemetry generated by instrumentation.
type TraceHandler interface {
	// HandleTrace handles a batch of trace telemetry produced by
	// auto-instrumentation for a single scope and conforming to the semantic
	// convention schema url.
	//
	// This method needs to be fast. The auto-instrumentation calls this method
	// in the hot-path of telemetry generation. Queueing or batching and
	// asynchronous processing should be utilized by the Handler to ensure this
	// method responds as fast as possible.
	HandleTrace(scope pcommon.InstrumentationScope, url string, spans ptrace.SpanSlice)
}

// MetricHandler handles metric telemetry generated by instrumentation.
type MetricHandler interface {
	// HandleMetric handles a batch of metric telemetry produced by
	// auto-instrumentation for a single scope and conforming to the semantic
	// convention schema url.
	//
	// This method needs to be fast. The auto-instrumentation calls this method
	// in the hot-path of telemetry generation. Queueing or batching and
	// asynchronous processing should be utilized by the Handler to ensure this
	// method responds as fast as possible.
	HandleMetric(scope pcommon.InstrumentationScope, url string, metrics pmetric.MetricSlice)
}

// LogHandler handles log telemetry generated by instrumentation.
type LogHandler interface {
	// HandleLog handles a batch of log telemetry produced by
	// auto-instrumentation for a single scope and conforming to the semantic
	// convention schema url.
	//
	// This method needs to be fast. The auto-instrumentation calls this method
	// in the hot-path of telemetry generation. Queueing or batching and
	// asynchronous processing should be utilized by the Handler to ensure this
	// method responds as fast as possible.
	HandleLog(scope pcommon.InstrumentationScope, url string, logs plog.LogRecordSlice)
}
