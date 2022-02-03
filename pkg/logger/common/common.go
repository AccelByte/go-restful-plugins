// Copyright 2018-2019 AccelByte Inc
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

package common

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/AccelByte/go-restful-plugins/v4/pkg/auth/iam"
	"github.com/AccelByte/go-restful-plugins/v4/pkg/constant"
	"github.com/AccelByte/go-restful-plugins/v4/pkg/logger/log"
	"github.com/AccelByte/go-restful-plugins/v4/pkg/trace"
	"github.com/AccelByte/go-restful-plugins/v4/pkg/util"
	publicsourceip "github.com/AccelByte/public-source-ip"
	"github.com/emicklei/go-restful/v3"
	"github.com/sirupsen/logrus"
)

var (
	FullAccessLogEnabled               bool
	FullAccessLogSupportedContentTypes []string
	FullAccessLogMaxBodySize           int

	fullAccessLogLogger *logrus.Logger
)

const (
	commonLogFormat     = `%s - %s [%s] "%s %s %s" %d %d %d`
	fullAccessLogFormat = `time=%s log_type=access method=%s path="%s" status=%d duration=%d length=%d source_ip=%s user_agent="%s" referer="%s" trace_id=%s namespace=%s user_id=%s client_id=%s request_content_type="%s" request_body=AB[%s]AB response_content_type="%s" response_body=AB[%s]AB`
)

// fullAccessLogFormatter represent logrus.Formatter,
// this is used to print the custom format for access log.
type fullAccessLogFormatter struct {
}

func (f *fullAccessLogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	return []byte(entry.Message + "\n"), nil
}

func init() {
	if s, exists := os.LookupEnv("FULL_ACCESS_LOG_ENABLED"); exists {
		value, err := strconv.ParseBool(s)
		if err != nil {
			logrus.Errorf("Parse FULL_ACCESS_LOG_ENABLED env error: %v", err)
		}
		FullAccessLogEnabled = value
	}

	if s, exists := os.LookupEnv("FULL_ACCESS_LOG_SUPPORTED_CONTENT_TYPES"); exists {
		FullAccessLogSupportedContentTypes = strings.Split(s, ",")
	} else {
		FullAccessLogSupportedContentTypes = []string{"application/json", "application/xml", "application/x-www-form-urlencoded", "text/plain", "text/html"}
	}

	FullAccessLogMaxBodySize = 10 << 10 // 10KB
	if s, exists := os.LookupEnv("FULL_ACCESS_LOG_MAX_BODY_SIZE"); exists {
		value, err := strconv.ParseInt(s, 0, 64)
		if err != nil {
			logrus.Errorf("Parse FULL_ACCESS_LOG_MAX_BODY_SIZE env error: %v", err)
		}
		FullAccessLogMaxBodySize = int(value)
	}
}

// Log is a filter that will log incoming request into the defined Log format
func Log(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	if FullAccessLogEnabled {
		fullAccessLogFilter(req, resp, chain)
	} else {
		simpleAccessLogFilter(req, resp, chain)
	}
}

// simpleAccessLogFilter will print the access log in simple common log format
func simpleAccessLogFilter(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	start := time.Now()
	username := "-"

	if req.Request.URL.User != nil {
		if name := req.Request.URL.User.Username(); name != "" {
			username = name
		}
	}

	chain.ProcessFilter(req, resp)

	duration := time.Since(start)
	logrus.Infof(commonLogFormat,
		publicsourceip.PublicIP(&http.Request{Header: req.Request.Header}),
		username,
		time.Now().Format("02/Jan/2006:15:04:05 -0700"),
		req.Request.Method,
		req.Request.URL.RequestURI(),
		req.Request.Proto,
		resp.StatusCode(),
		resp.ContentLength(),
		duration.Milliseconds(),
	)
}

// fullAccessLogFilter will print the access log in complete log format
func fullAccessLogFilter(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	// initialize custom logger for full access log
	if fullAccessLogLogger == nil {
		fullAccessLogLogger = &logrus.Logger{
			Out:       os.Stdout,
			Level:     logrus.GetLevel(),
			Formatter: &fullAccessLogFormatter{},
		}
	}

	start := time.Now()

	sourceIP := publicsourceip.PublicIP(&http.Request{Header: req.Request.Header})
	referer := req.HeaderParameter(constant.Referer)
	userAgent := req.HeaderParameter(constant.UserAgent)
	requestContentType := req.HeaderParameter(constant.ContentType)
	requestBody := getRequestBody(req, requestContentType)

	// decorate the original http.ResponseWriter with ResponseWriterInterceptor so we can intercept to get the response bytes
	respWriterInterceptor := &ResponseWriterInterceptor{ResponseWriter: resp.ResponseWriter}
	resp.ResponseWriter = respWriterInterceptor

	chain.ProcessFilter(req, resp)

	responseContentType := respWriterInterceptor.Header().Get(constant.ContentType)
	responseBody := getResponseBody(respWriterInterceptor, responseContentType)
	traceID := req.Attribute(trace.TraceIDKey)

	var tokenNamespace, tokenUserID, tokenClientID string
	if jwtClaims := iam.RetrieveJWTClaims(req); jwtClaims != nil {
		tokenNamespace = jwtClaims.Namespace
		tokenUserID = jwtClaims.Subject
		tokenClientID = jwtClaims.ClientID
	}

	requestUri := req.Request.URL.RequestURI()

	// mask sensitive field(s) in query params, request body and response body
	if maskedQueryParams := req.Attribute(log.MaskedQueryParams); maskedQueryParams != nil {
		requestUri = log.MaskQueryParams(requestUri, maskedQueryParams.(string))
	}
	if maskedRequestFields := req.Attribute(log.MaskedRequestFields); maskedRequestFields != nil && requestBody != "" {
		requestBody = log.MaskFields(requestContentType, requestBody, maskedRequestFields.(string))
	}
	if maskedResponseFields := req.Attribute(log.MaskedResponseFields); maskedResponseFields != nil && responseBody != "" {
		responseBody = log.MaskFields(responseContentType, responseBody, maskedResponseFields.(string))
	}

	duration := time.Since(start)

	fullAccessLogLogger.Infof(fullAccessLogFormat,
		time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		req.Request.Method,
		requestUri,
		resp.StatusCode(),
		duration.Milliseconds(),
		resp.ContentLength(),
		sourceIP,
		userAgent,
		referer,
		traceID,
		tokenNamespace,
		tokenUserID,
		tokenClientID,
		requestContentType,
		requestBody,
		responseContentType,
		responseBody,
	)
}

// getRequestBody will get the request body from Request object
func getRequestBody(req *restful.Request, contentType string) string {
	if contentType == "" || !isSupportedContentType(contentType) {
		return ""
	}

	bodyBytes, err := ioutil.ReadAll(req.Request.Body)
	if err != nil {
		logrus.Errorf("failed to read request body: %v", err.Error())
	}
	if len(bodyBytes) != 0 {
		// set the original bytes back into request body reader
		req.Request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

		if len(bodyBytes) > FullAccessLogMaxBodySize {
			return "data too large"
		}

		if strings.Contains(contentType, "application/json") {
			return util.MinifyJSON(bodyBytes)
		}
		return string(bodyBytes)
	}
	return ""
}

// getResponseBody will get the response body from ResponseWriterInterceptor object
func getResponseBody(respWriter *ResponseWriterInterceptor, contentType string) string {
	if contentType == "" || !isSupportedContentType(contentType) {
		return ""
	}

	if len(respWriter.data) > FullAccessLogMaxBodySize {
		return "data too large"
	}

	if strings.Contains(contentType, "application/json") {
		return util.MinifyJSON(respWriter.data)
	}
	return string(respWriter.data)
}

func isSupportedContentType(contentType string) bool {
	for _, v := range FullAccessLogSupportedContentTypes {
		if strings.Contains(contentType, v) {
			return true
		}
	}
	return false
}
