// Copyright 2021 AccelByte Inc
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

package constant

// HTTP header names.
const (
	ContentType         = "Content-Type"
	Referer             = "Referer"
	UserAgent           = "User-Agent"
	GameClientVersion   = "Game-Client-Version"
	AccelByteSDKVersion = "AccelByte-SDK-Version"
	AccelByteOSSVersion = "AccelByte-OSS-Version"
	FlightID            = "x-flight-id"
	SessionID           = "X-Ab-SessionID"
)

// Log field names.
const (
	// TraceID is the name of the log field that holds a request's OpenTelemetry trace ID.
	TraceID = "trace_id"
)

// Environment variable defaults.
const (
	// DefaultRealmLive is the default for REALM_LIVE: the comma-separated realm names that
	// count as production, where detailed insufficient-permission errors are turned off.
	DefaultRealmLive = "prod,live,demo,trial"
)

// Permission actions.
const (
	PermissionRead   = "READ"
	PermissionCreate = "CREATE"
	PermissionUpdate = "UPDATE"
	PermissionDelete = "DELETE"
)
