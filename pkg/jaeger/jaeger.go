/*
 * Copyright 2019-2020 AccelByte Inc
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

package jaeger

import (
	"context"
	"fmt"

	"github.com/AccelByte/go-restful-plugins/v3/pkg/trace"
	"github.com/emicklei/go-restful"
)

type contextKeyType string

const (
	spanContextKey = contextKeyType("span")
)

func Filter() restful.FilterFunction {
	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		traceID := req.HeaderParameter(trace.TraceIDKey)

		span, ctx := StartSpan(req, "Request "+req.Request.Method+" "+req.Request.URL.Path)
		span.SetTag(trace.TraceIDKey, traceID)
		defer Finish(span)

		ctx = context.WithValue(ctx, spanContextKey, span)

		req.Request = req.Request.WithContext(ctx)

		chain.ProcessFilter(req, resp)

		AddLog(span, "Response status code", fmt.Sprintf("%v", resp.StatusCode()))
	}
}
