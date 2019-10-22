/*
 * Copyright 2018-2019 AccelByte Inc
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

package common

import (
	"net/http"
	"time"

	publicsourceip "github.com/AccelByte/public-source-ip"
	"github.com/emicklei/go-restful"
	"github.com/sirupsen/logrus"
)

// Log is a filter that will log incoming request with Common Log Format
func Log(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	username := "-"
	if req.Request.URL.User != nil {
		if name := req.Request.URL.User.Username(); name != "" {
			username = name
		}
	}
	chain.ProcessFilter(req, resp)
	logrus.Infof(`%s - %s [%s] "%s %s %s" %d %d`,
		publicsourceip.PublicIP(&http.Request{Header: req.Request.Header}),
		username,
		time.Now().Format("02/Jan/2006:15:04:05 -0700"),
		req.Request.Method,
		req.Request.URL.RequestURI(),
		req.Request.Proto,
		resp.StatusCode(),
		resp.ContentLength(),
	)
}
