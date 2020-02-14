package jaeger

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/AccelByte/go-restful-plugins/v3/pkg/logger/event"
	"github.com/emicklei/go-restful"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
	"github.com/sirupsen/logrus"
	jaegerclientgo "github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/transport"
	"github.com/uber/jaeger-client-go/zipkin"
)

var forwardHeaders = [...]string{
	"x-request-id",
	"x-ot-span-context",
	"x-cloud-trace-context",
	"traceparent",
	"grpc-trace-bin",
}

// InitGlobalTracer initialize global tracer
// Must be called in main function
func InitGlobalTracer(
	jaegerAgentHost string,
	jaegerCollectorEndpoint string,
	serviceName string,
	realm string,
) io.Closer {
	zipkinPropagator := zipkin.NewZipkinB3HTTPHeaderPropagator()
	injector := jaegerclientgo.TracerOptions.Injector(opentracing.HTTPHeaders, zipkinPropagator)
	extractor := jaegerclientgo.TracerOptions.Extractor(opentracing.HTTPHeaders, zipkinPropagator)

	// Zipkin shares span ID between client and server spans; it must be enabled via the following option.
	zipkinSharedRPCSpan := jaegerclientgo.TracerOptions.ZipkinSharedRPCSpan(true)
	var reporter jaegerclientgo.Reporter
	if jaegerAgentHost == "" && jaegerCollectorEndpoint == "" {
		reporter = jaegerclientgo.NewNullReporter() // for running locally
		logrus.Info("Jaeger client configured to be silent")
	} else {
		var sender jaegerclientgo.Transport
		if jaegerCollectorEndpoint != "" {
			sender = transport.NewHTTPTransport(jaegerCollectorEndpoint)
			logrus.Infof("Jaeger client configured to use the collector: %s", jaegerCollectorEndpoint)
		} else {
			var err error
			sender, err = jaegerclientgo.NewUDPTransport(jaegerAgentHost, 0)
			if err != nil {
				logrus.Errorf("Jaeger transport initialization error: %s", err.Error())
			}
			logrus.Infof("Jaeger client configured to use the agent: %s", jaegerAgentHost)
		}

		reporter = jaegerclientgo.NewRemoteReporter(
			sender,
			jaegerclientgo.ReporterOptions.BufferFlushInterval(1*time.Second),
			jaegerclientgo.ReporterOptions.Logger(jaegerclientgo.StdLogger),
		)
	}

	newTracer, closer := jaegerclientgo.NewTracer(
		serviceName+"."+realm,
		jaegerclientgo.NewConstSampler(true),
		reporter,
		injector,
		extractor,
		zipkinSharedRPCSpan,
		jaegerclientgo.TracerOptions.PoolSpans(false),
	)
	// Set the singleton opentracing.Tracer with the Jaeger tracer.
	opentracing.SetGlobalTracer(newTracer)
	return closer
}

// InjectTrace to inject request header with context from current span
// Span returned here must be finished with span.finish()
// Any span not finished will not be sent to jaeger agent
func InjectTrace(ctx context.Context, incomingReq *restful.Request,
	outgoingReq *http.Request) (*http.Request, opentracing.Span, context.Context) {
	span, newCtx := StartSpanFromContext(ctx, "outgoing request")
	if span != nil {
		ext.HTTPUrl.Set(span, outgoingReq.Host+outgoingReq.RequestURI)
		ext.HTTPMethod.Set(span, outgoingReq.Method)
		_ = span.Tracer().Inject(
			span.Context(),
			opentracing.HTTPHeaders,
			opentracing.HTTPHeadersCarrier(outgoingReq.Header))
		for _, header := range forwardHeaders {
			if value := incomingReq.Request.Header.Get(header); value != "" {
				outgoingReq.Header.Set(header, value)
			}
		}
	} else {
		return outgoingReq, nil, nil
	}

	if logrus.GetLevel() >= logrus.DebugLevel {
		var header = make(map[string]string)
		for key, val := range outgoingReq.Header {
			key = strings.ToLower(key)
			if !strings.Contains(key, "auth") {
				header[key] = val[0]
			}
		}
		logrus.Debug("outgoing header : ", header)
	}

	if abTraceID := incomingReq.Request.Header.Get(event.TraceIDKey); abTraceID != "" {
		outgoingReq.Header.Set(event.TraceIDKey, abTraceID)
	}
	return outgoingReq, span, newCtx
}

// StartSpan to start a new child span from restful.Request
// Span returned here must be finished with span.finish()
// Any span not finished will not be sent to jaeger agent
func StartSpan(req *restful.Request, operationName string) (opentracing.Span, context.Context) {
	if logrus.GetLevel() >= logrus.DebugLevel {
		var header = make(map[string]string)
		for key, val := range req.Request.Header {
			key = strings.ToLower(key)
			if !strings.Contains(key, "auth") {
				header[key] = val[0]
			}
		}
		logrus.Debug("incoming header : ", header)
	}
	spanContext, err := ExtractRequestHeader(req)
	var span opentracing.Span
	if err != nil {
		logrus.Debug("request has no tracing context: ", err.Error())
		span = opentracing.StartSpan(operationName)
	} else {
		span = opentracing.StartSpan(
			operationName,
			opentracing.ChildOf(spanContext),
		)
	}
	ext.HTTPMethod.Set(span, req.Request.Method)
	ext.HTTPUrl.Set(span, req.Request.Host+req.Request.RequestURI)

	if abTraceID := req.Request.Header.Get(event.TraceIDKey); abTraceID != "" {
		AddTag(span, event.TraceIDKey, abTraceID)
	}
	return span, opentracing.ContextWithSpan(req.Request.Context(), span)
}

// StartSpanIfParentSpanExist to start a new child span from restful.Request if it contain SpanContext
// For example this function can be used in healthz endpoint,when we want to omit request from kubernetes liveness probe
// Span returned here must be finished with span.finish()
// Any span not finished will not be sent to jaeger agent
func StartSpanIfParentSpanExist(req *restful.Request, operationName string) (opentracing.Span, context.Context) {
	if logrus.GetLevel() >= logrus.DebugLevel {
		var header = make(map[string]string)
		for key, val := range req.Request.Header {
			key = strings.ToLower(key)
			if !strings.Contains(key, "auth") {
				header[key] = val[0]
			}
		}
		logrus.Debug("incoming header : ", header)
	}
	spanContext, err := ExtractRequestHeader(req)
	if err != nil {
		return nil, nil
	}
	span := opentracing.StartSpan(
		operationName,
		opentracing.ChildOf(spanContext),
	)
	ext.HTTPMethod.Set(span, req.Request.Method)
	ext.HTTPUrl.Set(span, req.Request.Host+req.Request.RequestURI)

	if abTraceID := req.Request.Header.Get(event.TraceIDKey); abTraceID != "" {
		AddTag(span, event.TraceIDKey, abTraceID)
	}
	return span, opentracing.ContextWithSpan(req.Request.Context(), span)
}

// StartDBSpan start DBSpan from context.
// Span returned here must be finished with span.finish()
// Any span not finished will not be sent to jaeger agent
func StartDBSpan(ctx context.Context, operationName string) (opentracing.Span, context.Context) {
	return StartSpanFromContext(ctx, "DB-"+operationName)
}

// StartSpanFromContext start span from context if context != nil.
// Span returned here must be finished with span.finish()
// Any span not finished will not be sent to jaeger agent
func StartSpanFromContext(ctx context.Context, operationName string) (opentracing.Span, context.Context) {
	if ctx != nil {
		childSpan, childCtx := opentracing.StartSpanFromContext(ctx, operationName)
		return childSpan, childCtx
	}
	return nil, ctx
}

func StartChildSpan(span opentracing.Span, name string) opentracing.Span {
	if span != nil {
		return opentracing.StartSpan(
			name,
			opentracing.ChildOf(span.Context()),
		)
	}

	return nil
}

func InjectSpanIntoRequest(span opentracing.Span, req *http.Request) {
	if span != nil {
		opentracing.GlobalTracer().Inject(
			span.Context(),
			opentracing.HTTPHeaders,
			opentracing.HTTPHeadersCarrier(req.Header))
	}
}

// ExtractRequestHeader to extract SpanContext from request header
func ExtractRequestHeader(req *restful.Request) (opentracing.SpanContext, error) {
	return opentracing.GlobalTracer().Extract(
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(req.Request.Header))
}

// Finish to finish span if it is exist
func Finish(span opentracing.Span) {
	if span != nil {
		span.Finish()
	}
}

// AddLog to add string log in span if span is valid
func AddLog(span opentracing.Span, key string, value string) {
	if span != nil {
		span.LogFields(log.String(key, value))
	}
}

// AddTag to add tag in span if span is valid
func AddTag(span opentracing.Span, key string, value string) {
	if span != nil {
		span.SetTag(key, value)
	}
}

// AddBaggage to add baggage in span if span is valid
// sets a key:value pair on this Span and its SpanContext
// that also propagates to descendants of this Span.
func AddBaggage(span opentracing.Span, key string, value string) {
	if span != nil {
		span.SetBaggageItem(key, value)
	}
}

// TraceError sends a log and a tag with Error into tracer
func TraceError(span opentracing.Span, err error) {
	AddLog(span, "Error", err.Error())
	AddTag(span, "Error", "true")
}

// TraceSQLQuery sends a log with SQL query into tracer
func TraceSQLQuery(span opentracing.Span, query string) {
	AddLog(span, "SQL", query)
}

// GetSpanFromRestfulContext get crated by jaeger Filter span from the context
func GetSpanFromRestfulContext(ctx context.Context) opentracing.Span {
	if span, ok := ctx.Value(spanContextKey).(opentracing.Span); ok {
		return span
	}

	logrus.Info("missed initialization of restful plugin jaeger.Filter")
	span, _ := StartSpanFromContext(ctx, "unnamed")

	return span
}
