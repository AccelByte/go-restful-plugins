// Copyright 2022 AccelByte Inc
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

package log

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/AccelByte/go-restful-plugins/v4/pkg/auth/iam"
	"github.com/AccelByte/go-restful-plugins/v4/pkg/constant"
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
	FullAccessLogRequestBodyEnabled    bool
	FullAccessLogResponseBodyEnabled   bool

	fullAccessLogLogger *logrus.Logger
)

const (
	fullAccessLogFormat = `time=%s log_type=access method=%s path="%s" status=%d duration=%d length=%d source_ip=%s user_agent="%s" referer="%s" trace_id=%s namespace=%s user_id=%s client_id=%s request_content_type="%s" request_body=AB[%s]AB req_body_size_kb=%.3f response_content_type="%s" response_body=AB[%s]AB resp_body_size_kb=%.3f operation="%s" flight_id="%s" game_version="%s" sdk_version="%s" oss_version="%s"`

	kb = 1 << 10
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

	FullAccessLogRequestBodyEnabled = true
	if s, exists := os.LookupEnv("FULL_ACCESS_LOG_REQUEST_BODY_ENABLED"); exists {
		value, err := strconv.ParseBool(s)
		if err != nil {
			logrus.Errorf("Parse FULL_ACCESS_LOG_REQUEST_BODY_ENABLED env error: %v", err)
		}
		FullAccessLogRequestBodyEnabled = value
	}

	FullAccessLogResponseBodyEnabled = true
	if s, exists := os.LookupEnv("FULL_ACCESS_LOG_RESPONSE_BODY_ENABLED"); exists {
		value, err := strconv.ParseBool(s)
		if err != nil {
			logrus.Errorf("Parse FULL_ACCESS_LOG_RESPONSE_BODY_ENABLED env error: %v", err)
		}
		FullAccessLogResponseBodyEnabled = value
	}

	fullAccessLogLogger = &logrus.Logger{
		Out:       os.Stdout,
		Level:     logrus.GetLevel(),
		Formatter: &fullAccessLogFormatter{},
	}
}

// AccessLog is a filter that will log incoming request into the Access Log format
func AccessLog(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	start := time.Now()

	sourceIP := publicsourceip.PublicIP(&http.Request{Header: req.Request.Header})
	referer := req.HeaderParameter(constant.Referer)
	userAgent := req.HeaderParameter(constant.UserAgent)
	flightID := req.HeaderParameter(constant.FlightID)
	gameClientVersion := req.HeaderParameter(constant.GameClientVersion)
	abSDKVersion := req.HeaderParameter(constant.AccelByteSDKVersion)
	abOSSVersion := req.HeaderParameter(constant.AccelByteOSSVersion)
	requestContentType := req.HeaderParameter(constant.ContentType)
	requestBody := "-"
	var requestBodySizeInKB float32 = 0

	requestUri := req.Request.URL.RequestURI()
	// mask sensitive field(s)
	if maskedQueryParams := req.Attribute(MaskedQueryParamsAttribute); maskedQueryParams != nil {
		requestUri = MaskQueryParams(requestUri, maskedQueryParams.(string))
	}

	requestBody, requestBodySizeInKB = getRequestBody(req, requestContentType, requestUri)

	// decorate the original http.ResponseWriter with ResponseWriterInterceptor so we can intercept to get the response bytes
	respWriterInterceptor := &ResponseWriterInterceptor{ResponseWriter: resp.ResponseWriter}
	resp.ResponseWriter = respWriterInterceptor

	chain.ProcessFilter(req, resp)

	var tokenNamespace, tokenUserID, tokenClientID string
	if val := req.Attribute(NamespaceAttribute); val != nil {
		tokenNamespace = val.(string)
	}
	if val := req.Attribute(UserIDAttribute); val != nil {
		tokenUserID = val.(string)
	}
	if val := req.Attribute(ClientIDAttribute); val != nil {
		tokenClientID = val.(string)
	}
	if jwtClaims := iam.RetrieveJWTClaims(req); jwtClaims != nil {
		// if tokenNamespace, tokenUserID or tokenClientID is empty,
		// fallback get from jwt claims
		if tokenNamespace == "" {
			tokenNamespace = jwtClaims.Namespace
		}
		if tokenUserID == "" {
			tokenUserID = jwtClaims.Subject
		}
		if tokenClientID == "" {
			tokenClientID = jwtClaims.ClientID
		}
	}

	responseContentType := respWriterInterceptor.Header().Get(constant.ContentType)
	responseBody := "-"
	var responseBodySizeInKB float32 = 0

	responseBody, responseBodySizeInKB = getResponseBody(respWriterInterceptor, responseContentType, requestUri)

	if FullAccessLogEnabled {
		if FullAccessLogRequestBodyEnabled {
			// mask sensitive field(s)
			// notes: we masked the request body after calling chain.ProcessFilter first,
			//        since the MaskedRequestFields attribute is initialized in the inner filter.
			if maskedRequestFields := req.Attribute(MaskedRequestFieldsAttribute); maskedRequestFields != nil && requestBody != "" {
				requestBody = MaskFields(requestContentType, requestBody, maskedRequestFields.(string))
			}
		}

		if FullAccessLogResponseBodyEnabled {
			// mask sensitive field(s)
			if maskedResponseFields := req.Attribute(MaskedResponseFieldsAttribute); maskedResponseFields != nil && responseBody != "" {
				responseBody = MaskFields(responseContentType, responseBody, maskedResponseFields.(string))
			}
		}
	}

	// extract operation id
	operation := ""
	selectedRoute := req.SelectedRoute()
	if selectedRoute != nil {
		operation = selectedRoute.Operation()
	}

	traceID := req.Attribute(trace.TraceIDKey)
	if traceID == nil {
		traceID = ""
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
		requestBodySizeInKB,
		responseContentType,
		responseBody,
		responseBodySizeInKB,
		operation,
		flightID,
		gameClientVersion,
		abSDKVersion,
		abOSSVersion,
	)
}

// getRequestBody will get the request body from Request object
func getRequestBody(req *restful.Request, contentType, requestURL string) (string, float32) {
	if contentType == "" || !isSupportedContentType(contentType) || req.Request == nil || req.Request.Body == nil {
		return "", 0
	}

	bodyBytes, err := io.ReadAll(req.Request.Body)
	if err != nil {
		logrus.Errorf("failed to read request body: %v", err.Error())
		return "", 0
	}
	if len(bodyBytes) != 0 {
		// set the original bytes back into request body reader
		req.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		requestBodySize := len(bodyBytes)
		requestBodySizeInKB := float32(requestBodySize) / kb

		if !FullAccessLogEnabled || !FullAccessLogRequestBodyEnabled {
			return "-", requestBodySizeInKB
		}

		if requestBodySize > FullAccessLogMaxBodySize {
			return "data too large", requestBodySizeInKB
		}

		if strings.Contains(contentType, "application/json") {
			mJson, mErr := util.MinifyJSON(bodyBytes)
			if mErr != nil {
				logrus.Infof("failed to minify request body json, error: %v, source:%s, %s", mErr.Error(), mJson, requestURL)
			}
			return mJson, requestBodySizeInKB
		}

		bodyString := string(bodyBytes)
		bodyString = strings.ReplaceAll(bodyString, "\n", "\\n")
		bodyString = strings.ReplaceAll(bodyString, "\r", "\\r")
		return bodyString, requestBodySizeInKB
	}
	return "", 0
}

// getResponseBody will get the response body from ResponseWriterInterceptor object
func getResponseBody(respWriter *ResponseWriterInterceptor, contentType, requestURL string) (string, float32) {
	if contentType == "" || !isSupportedContentType(contentType) {
		return "", 0
	}

	responseBodySize := len(respWriter.data)
	responseBodySizeInKB := float32(responseBodySize) / kb

	if !FullAccessLogEnabled || !FullAccessLogResponseBodyEnabled {
		return "-", responseBodySizeInKB
	}

	if responseBodySize > FullAccessLogMaxBodySize {
		return "data too large", responseBodySizeInKB
	}

	if strings.Contains(contentType, "application/json") {
		mJson, mErr := util.MinifyJSON(respWriter.data)
		if mErr != nil {
			logrus.Warnf("failed to minify response body json, error: %v, source:%s, %s", mErr.Error(), mJson, requestURL)
		}
		return mJson, responseBodySizeInKB
	}

	bodyString := string(respWriter.data)
	bodyString = strings.ReplaceAll(bodyString, "\n", "\\n")
	bodyString = strings.ReplaceAll(bodyString, "\r", "\\r")
	return bodyString, responseBodySizeInKB
}

func isSupportedContentType(contentType string) bool {
	for _, v := range FullAccessLogSupportedContentTypes {
		if strings.Contains(contentType, v) {
			return true
		}
	}
	return false
}
