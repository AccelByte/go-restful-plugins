/*
 * Copyright 2019 AccelByte Inc
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

package trace

import (
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/emicklei/go-restful"
	"github.com/google/uuid"
)

const (
	TraceIDKey = "X-Ab-TraceID"
)

func Filter() restful.FilterFunction {
	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		if req.HeaderParameter(TraceIDKey) == "" {
			traceID, err := generateUUID()
			if err != nil {
				logrus.Errorf("Unable to generate UUID %s", err.Error())
			}
			req.Request.Header.Add(TraceIDKey, fmt.Sprintf("%x-%s", time.Now().UTC().Unix(), traceID))
		}
		chain.ProcessFilter(req, resp)
	}
}

func generateUUID() (string, error) {
	newUUID, err := uuid.NewRandom()
	return strings.Replace(newUUID.String(), "-", "", -1), err
}
