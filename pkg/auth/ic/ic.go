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

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/AccelByte/ic-go-sdk"
	"github.com/emicklei/go-restful/v3"
	"net/http"
	"os"
	"strings"

	"github.com/AccelByte/go-restful-plugins/v4/pkg/constant"
	"github.com/sirupsen/logrus"
)

const (
	// ClaimsAttribute is the key for JWT claims stored in the request
	ClaimsAttribute = "ICJWTClaims"

	accessTokenCookieKey = "access_token"
	tokenFromCookie      = "cookie"
	tokenFromHeader      = "header"
)

var DevStackTraceable bool

// FilterOption extends the basic auth filter functionality
type FilterOption func(req *restful.Request, icClient ic.Client, claims *ic.JWTClaims) error

// FilterInitializationOptions hold options for Filter during initialization
type FilterInitializationOptions struct {
}

// Filter handles auth using filter
type Filter struct {
	icClient ic.Client
	options  *FilterInitializationOptions
}

// ErrorResponse is the generic structure for communicating errors from a REST endpoint.
type ErrorResponse struct {
	ErrorCode    int    `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
}

// NewFilter creates new Filter instance
func NewFilter(client ic.Client) *Filter {
	return &Filter{icClient: client}
}

// Auth returns a filter that filters request with valid access token in auth header or cookie
// The token's claims will be passed in the request.attributes["ICJWTClaims"] = *ic.JWTClaims{}
// This filter is expandable through FilterOption parameter
// Example:
// ic.Auth(
//
//	WithValidUser(),
//	WithPermission("ADMIN"),
//
// )
func (filter *Filter) Auth(opts ...FilterOption) restful.FilterFunction {
	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		token, _, err := parseAccessToken(req)
		if err != nil {
			logrus.Warn("unauthorized access: ", err)
			logIfErr(resp.WriteHeaderAndJson(http.StatusUnauthorized, ErrorResponse{
				ErrorCode:    UnauthorizedAccess,
				ErrorMessage: ErrorCodeMapping[UnauthorizedAccess],
			}, restful.MIME_JSON))

			return
		}

		claims, err := filter.icClient.ValidateAndParseClaims(token)
		if err != nil {
			logrus.Warn("unauthorized access: ", err)
			if err.Error() == ErrorCodeMapping[TokenIsExpired] {
				logIfErr(resp.WriteHeaderAndJson(http.StatusUnauthorized, ErrorResponse{
					ErrorCode:    TokenIsExpired,
					ErrorMessage: ErrorCodeMapping[TokenIsExpired],
				}, restful.MIME_JSON))
				return
			}
			logIfErr(resp.WriteHeaderAndJson(http.StatusUnauthorized, ErrorResponse{
				ErrorCode:    UnauthorizedAccess,
				ErrorMessage: ErrorCodeMapping[UnauthorizedAccess],
			}, restful.MIME_JSON))
			return
		}

		req.SetAttribute(ClaimsAttribute, claims)
		for _, opt := range opts {
			if err = opt(req, filter.icClient, claims); err != nil {
				if svcErr, ok := err.(restful.ServiceError); ok {
					logrus.Warn(svcErr.Message)

					var respErr ErrorResponse

					err = json.Unmarshal([]byte(svcErr.Message), &respErr)
					if err == nil {
						logIfErr(resp.WriteHeaderAndJson(svcErr.Code, respErr, restful.MIME_JSON))
					} else {
						logIfErr(resp.WriteErrorString(svcErr.Code, svcErr.Message))
					}

					return
				}

				logrus.Warn(err)
				logIfErr(resp.WriteErrorString(http.StatusUnauthorized, err.Error()))

				return
			}
		}

		chain.ProcessFilter(req, resp)
	}
}

// PublicAuth returns a filter that allow unauthenticated request and request with valid access token in auth header or cookie
// If request has access token, the token's claims will be passed in the request.attributes["ICJWTClaims"] = *ic.JWTClaims{}
// If request has invalid access token, then request treated as public access without claims
// This filter is expandable through FilterOption parameter
// Example:
// ic.PublicAuth(
//
//	WithValidUser(),
//	WithPermission("ADMIN"),
//
// )
func (filter *Filter) PublicAuth(opts ...FilterOption) restful.FilterFunction {
	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		token, _, err := parseAccessToken(req)
		if err != nil {
			chain.ProcessFilter(req, resp)
			return
		}

		claims, err := filter.icClient.ValidateAndParseClaims(token)
		if err != nil {
			logrus.Warn("unauthorized access for public endpoint: ", err)
			chain.ProcessFilter(req, resp)
			return
		}

		req.SetAttribute(ClaimsAttribute, claims)
		for _, opt := range opts {
			if err = opt(req, filter.icClient, claims); err != nil {
				logrus.Warn(err)
				req.SetAttribute(ClaimsAttribute, nil)
				chain.ProcessFilter(req, resp)
				return
			}
		}

		chain.ProcessFilter(req, resp)
	}
}

// RetrieveJWTClaims is a convenience function to retrieve JWT claims
// from restful.Request.
// Warning: the claims can be nil if the request wasn't filtered through Auth()
func RetrieveJWTClaims(request *restful.Request) *ic.JWTClaims {
	claims, _ := request.Attribute(ClaimsAttribute).(*ic.JWTClaims)
	return claims
}

// WithValidUser filters request with valid user only
func WithValidUser() FilterOption {
	return func(req *restful.Request, icClient ic.Client, claims *ic.JWTClaims) error {
		if claims.Subject == "" {
			return respondError(http.StatusForbidden, TokenIsNotUserToken,
				"access forbidden: "+ErrorCodeMapping[TokenIsNotUserToken])
		}

		return nil
	}
}

// WithPermission filters request with valid permission only
func WithPermission(permission *ic.Permission) FilterOption {
	return func(req *restful.Request, icClient ic.Client, claims *ic.JWTClaims) error {
		requiredPermissionResources := make(map[string]string)
		requiredPermissionResources["{userId}"] = req.PathParameter("userId")
		requiredPermissionResources["{organizationId}"] = req.PathParameter("organizationId")
		requiredPermissionResources["{projectId}"] = req.PathParameter("projectId")

		valid, err := icClient.ValidatePermission(claims, *permission, requiredPermissionResources)
		if err != nil {
			return respondError(http.StatusInternalServerError, InternalServerError,
				"unable to validate permission: "+err.Error())
		}

		insufficientPermissionMessage := ErrorCodeMapping[InsufficientPermissions]
		if DevStackTraceable {
			action := ActionConverter(permission.Action)
			insufficientPermissionMessage = fmt.Sprintf("%s. Required permission: %s [%s]", insufficientPermissionMessage,
				permission.Resource, action)
		}
		if !valid {
			return respondError(http.StatusForbidden, InsufficientPermissions,
				"access forbidden: "+insufficientPermissionMessage)
		}

		return nil
	}
}

// parseAccessToken is used to read token from Authorization Header or Cookie.
// it will return the token value and token from.
func parseAccessToken(request *restful.Request) (string, string, error) {
	authorization := request.HeaderParameter("Authorization")
	if strings.HasPrefix(authorization, "Bearer ") {
		if token := strings.TrimPrefix(authorization, "Bearer "); token != "" {
			return token, tokenFromHeader, nil
		}
	}

	for _, cookie := range request.Request.Cookies() {
		if cookie.Name == accessTokenCookieKey && cookie.Value != "" {
			return cookie.Value, tokenFromCookie, nil
		}
	}

	return "", "", errors.New("token not provided in request header")
}

func logIfErr(err error) {
	if err != nil {
		logrus.Error(err)
	}
}

func respondError(httpStatus, errorCode int, errorMessage string) restful.ServiceError {
	messageByte, err := json.Marshal(ErrorResponse{
		ErrorCode:    errorCode,
		ErrorMessage: errorMessage,
	})
	if err != nil {
		errMsgByte, _ := json.Marshal(ErrorResponse{
			ErrorCode:    InternalServerError,
			ErrorMessage: "unable to parse error message : " + err.Error(),
		})

		return restful.ServiceError{
			Code:    http.StatusInternalServerError,
			Message: string(errMsgByte),
		}
	}

	return restful.ServiceError{
		Code:    httpStatus,
		Message: string(messageByte),
	}
}

func init() {
	DevStackTraceable = true // activate verbose insufficient error message in non-prod environment
	realmName, realmNameExists := os.LookupEnv("REALM_NAME")
	if !realmNameExists {
		DevStackTraceable = false
		return
	}

	realmLive, realmLiveExists := os.LookupEnv("REALM_LIVE")
	if !realmLiveExists {
		realmLive = constant.DefaultRealmLive
	}

	realmLives := strings.Split(realmLive, ",")
	for _, rl := range realmLives {
		if realmName == rl {
			DevStackTraceable = false
			return
		}
	}
}

// ActionConverter convert IC action bit to human-readable
func ActionConverter(action int) string {
	var ActionStr string
	switch action {
	case ic.ActionRead:
		ActionStr = constant.PermissionRead
	case ic.ActionCreate:
		ActionStr = constant.PermissionCreate
	case ic.ActionUpdate:
		ActionStr = constant.PermissionUpdate
	case ic.ActionDelete:
		ActionStr = constant.PermissionDelete
	default:
		return ""
	}
	return ActionStr
}
