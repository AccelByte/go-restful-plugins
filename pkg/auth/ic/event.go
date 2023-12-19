// Copyright 2023 AccelByte Inc
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

package ic

const (
	// Global Error Codes
	InternalServerError     = 20000
	UnauthorizedAccess      = 20001
	ForbiddenAccess         = 20002
	TokenIsExpired          = 20003
	InsufficientPermissions = 20004
	InsufficientScope       = 20005
	TokenIsNotUserToken     = 20006
)

var ErrorCodeMapping = map[int]string{
	// Global Error Codes
	InternalServerError:     "internal server error",
	UnauthorizedAccess:      "unauthorized access",
	ForbiddenAccess:         "forbidden access",
	InsufficientPermissions: "insufficient permissions",
	InsufficientScope:       "insufficient scope",
	TokenIsNotUserToken:     "token is not user token",
	TokenIsExpired:          "token is expired",
}
