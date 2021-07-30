// Copyright 2019 AccelByte Inc
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

package trace

import (
	"fmt"
	"strings"
	"time"

	"github.com/emicklei/go-restful/v3"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// IDType is the type for trace_id
type IDType string

const (
	TraceIDKey = "X-Ab-TraceID"

	SimpleTraceID    IDType = "Simple"    // format: uuid
	TimeBasedTraceID IDType = "TimeBased" // format: requestTime-uuid
)

func Filter() restful.FilterFunction {
	return initFilter(TimeBasedTraceID)
}

func FilterWithOption(traceIDType IDType) restful.FilterFunction {
	return initFilter(traceIDType)
}

func initFilter(traceIDType IDType) restful.FilterFunction {
	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		traceID := req.HeaderParameter(TraceIDKey)
		if traceID == "" {
			var err error
			traceID, err = generateUUID()
			if err != nil {
				logrus.Errorf("Unable to generate UUID %s", err.Error())
			}

			if traceIDType == TimeBasedTraceID {
				traceID = fmt.Sprintf("%x-%s", time.Now().UTC().Unix(), traceID)
			}
			req.Request.Header.Add(TraceIDKey, traceID)
		}

		req.SetAttribute(TraceIDKey, traceID)
		resp.Header().Set(TraceIDKey, traceID)

		chain.ProcessFilter(req, resp)
	}
}

func generateUUID() (string, error) {
	newUUID, err := uuid.NewRandom()
	return strings.ReplaceAll(newUUID.String(), "-", ""), err
}
