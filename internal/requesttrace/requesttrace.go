// Copyright 2026 AccelByte Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package requesttrace reads the OpenTelemetry trace ID of a go-restful request, which the access and
// event logs write as trace_id.
package requesttrace

import (
	"context"

	"github.com/emicklei/go-restful/v3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// TraceID returns the trace ID of the span in the request context, otherwise that of the trace context
// the request arrived with (read with the global propagator), otherwise "". It never falls back to
// X-Ab-TraceID.
//
// Call it after the filter chain, so it sees a span that a later filter put on the request, such as
// observability-go-sdk's request filter registered after the logging filter.
func TraceID(req *restful.Request) string {
	if req == nil || req.Request == nil {
		return ""
	}

	if spanContext := oteltrace.SpanContextFromContext(req.Request.Context()); spanContext.IsValid() {
		return spanContext.TraceID().String()
	}

	// Some services start their request span in the handler without putting it on the request.
	// That span continues the incoming trace, so the incoming trace ID is the same.
	ctx := otel.GetTextMapPropagator().Extract(context.Background(), propagation.HeaderCarrier(req.Request.Header))
	if spanContext := oteltrace.SpanContextFromContext(ctx); spanContext.IsValid() {
		return spanContext.TraceID().String()
	}

	return ""
}
