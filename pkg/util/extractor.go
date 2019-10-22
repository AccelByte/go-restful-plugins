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

package util

import (
	"github.com/AccelByte/go-restful-plugins/v3/pkg/auth/iam"
	"github.com/emicklei/go-restful"
)

const (
	traceIDKey   = "X-Ab-TraceID"
	sessionIDKey = "X-Ab-SessionID"
)

// ExtractDefault is default function for extracting attribute for filter event logger
func ExtractDefault(req *restful.Request) (userID string, clientID []string,
	namespace string, traceID string, sessionID string) {
	traceID = req.HeaderParameter(traceIDKey)
	sessionID = req.HeaderParameter(sessionIDKey)
	claims := iam.RetrieveJWTClaims(req)
	if claims != nil {
		return claims.Subject, []string{claims.ClientID}, claims.Namespace, traceID, sessionID
	}
	return "", []string{}, "", traceID, sessionID
}
