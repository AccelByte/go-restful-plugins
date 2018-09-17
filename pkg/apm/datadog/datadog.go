/*
 * Copyright 2018 AccelByte Inc
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package datadog

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/emicklei/go-restful"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

// Start initiates the tracer
func Start(addr string, serviceName string, environment string, debugMode bool) {
	tracer.Start(
		tracer.WithAgentAddr(addr),
		tracer.WithServiceName(serviceName),
		tracer.WithGlobalTag(ext.Environment, environment),
		tracer.WithDebugMode(debugMode))
}

// Trace is a filter that will trace incoming request
func Trace(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	opts := []ddtrace.StartSpanOption{
		tracer.ResourceName(req.SelectedRoutePath()),
		tracer.SpanType(ext.SpanTypeWeb),
		tracer.Tag(ext.HTTPMethod, req.Request.Method),
		tracer.Tag(ext.HTTPURL, req.Request.URL.Path),
	}
	if spanctx, err := tracer.Extract(tracer.HTTPHeadersCarrier(req.Request.Header)); err == nil {
		opts = append(opts, tracer.ChildOf(spanctx))
	}
	span, ctx := tracer.StartSpanFromContext(req.Request.Context(), "http.request", opts...)
	defer span.Finish()

	// pass the span through the request context
	req.Request = req.Request.WithContext(ctx)

	chain.ProcessFilter(req, resp)

	span.SetTag(ext.HTTPCode, strconv.Itoa(resp.StatusCode()))

	if resp.Error() != nil {
		span.SetTag(ext.Error, resp.Error())
	}
}

// Inject adds tracer header to a HTTP request
func Inject(outRequest *http.Request, restfulRequest *restful.Request) error {
	span, ok := tracer.SpanFromContext(restfulRequest.Request.Context())
	if !ok {
		return errors.New("no trace context in the request, request is not instrumented")
	}
	return tracer.Inject(span.Context(), tracer.HTTPHeadersCarrier(outRequest.Header))
}
