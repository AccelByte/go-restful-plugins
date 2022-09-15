// Copyright 2018 AccelByte Inc
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

package iam

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/AccelByte/go-restful-plugins/v4/pkg/auth/util"
	"github.com/AccelByte/go-restful-plugins/v4/pkg/constant"
	"github.com/AccelByte/iam-go-sdk"
	"github.com/emicklei/go-restful/v3"
	"github.com/sirupsen/logrus"
)

const (
	// ClaimsAttribute is the key for JWT claims stored in the request
	ClaimsAttribute = "JWTClaims"

	accessTokenCookieKey = "access_token"
	tokenFromCookie      = "cookie"
	tokenFromHeader      = "header"
)

// FilterOption extends the basic auth filter functionality
type FilterOption func(req *restful.Request, iamClient iam.Client, claims *iam.JWTClaims) error

// FilterInitializationOptions hold options for Filter during initialization
type FilterInitializationOptions struct {
	StrictRefererHeaderValidation              bool // Enable full path check of redirect uri in referer header validation
	AllowSubdomainMatchRefererHeaderValidation bool // Allow checking with subdomain
}

// Filter handles auth using filter
type Filter struct {
	iamClient iam.Client
	options   *FilterInitializationOptions
}

// ErrorResponse is the generic structure for communicating errors from a REST endpoint.
type ErrorResponse struct {
	ErrorCode    int    `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
}

// NewFilter creates new Filter instance
func NewFilter(client iam.Client) *Filter {
	return &Filter{iamClient: client, options: &FilterInitializationOptions{}}
}

// NewFilterWithOptions creates new Filter instance with Options
func NewFilterWithOptions(client iam.Client, options *FilterInitializationOptions) *Filter {
	if options == nil {
		return &Filter{iamClient: client, options: &FilterInitializationOptions{}}
	}
	return &Filter{iamClient: client, options: options}
}

// Auth returns a filter that filters request with valid access token in auth header or cookie
// The token's claims will be passed in the request.attributes["JWTClaims"] = *iam.JWTClaims{}
// This filter is expandable through FilterOption parameter
// Example:
// iam.Auth(
// 		WithValidUser(),
//		WithPermission("ADMIN"),
// )
func (filter *Filter) Auth(opts ...FilterOption) restful.FilterFunction {
	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		token, tokenFrom, err := parseAccessToken(req)
		if err != nil {
			logrus.Warn("unauthorized access: ", err)
			logIfErr(resp.WriteHeaderAndJson(http.StatusUnauthorized, ErrorResponse{
				ErrorCode:    UnauthorizedAccess,
				ErrorMessage: ErrorCodeMapping[UnauthorizedAccess],
			}, restful.MIME_JSON))

			return
		}

		claims, err := filter.iamClient.ValidateAndParseClaims(token)
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

		if tokenFrom == tokenFromCookie {
			valid := filter.validateRefererHeader(req, claims)
			if !valid {
				logIfErr(resp.WriteHeaderAndJson(http.StatusUnauthorized, ErrorResponse{
					ErrorCode:    InvalidRefererHeader,
					ErrorMessage: ErrorCodeMapping[InvalidRefererHeader],
				}, restful.MIME_JSON))

				return
			}
		}

		for _, opt := range opts {
			if err = opt(req, filter.iamClient, claims); err != nil {
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

// PublicAuth returns a filter that allow unauthenticate request and request with valid access token in auth header or cookie
// If request has acces token, the token's claims will be passed in the request.attributes["JWTClaims"] = *iam.JWTClaims{}
// If request has invalid access token, then request treated as public access without claims
// This filter is expandable through FilterOption parameter
// Example:
// iam.PublicAuth(
// 		WithValidUser(),
//		WithPermission("ADMIN"),
// )
func (filter *Filter) PublicAuth(opts ...FilterOption) restful.FilterFunction {
	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		token, tokenFrom, err := parseAccessToken(req)
		if err != nil {
			chain.ProcessFilter(req, resp)
			return
		}

		claims, err := filter.iamClient.ValidateAndParseClaims(token)
		if err != nil {
			logrus.Warn("unauthorized access for public endpoint: ", err)
			chain.ProcessFilter(req, resp)
			return
		}

		req.SetAttribute(ClaimsAttribute, claims)

		if tokenFrom == tokenFromCookie {
			valid := filter.validateRefererHeader(req, claims)
			if !valid {
				req.SetAttribute(ClaimsAttribute, nil)
				chain.ProcessFilter(req, resp)
				return
			}
		}

		for _, opt := range opts {
			if err = opt(req, filter.iamClient, claims); err != nil {
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
func RetrieveJWTClaims(request *restful.Request) *iam.JWTClaims {
	claims, _ := request.Attribute(ClaimsAttribute).(*iam.JWTClaims)
	return claims
}

// WithValidUser filters request with valid user only
func WithValidUser() FilterOption {
	return func(req *restful.Request, iamClient iam.Client, claims *iam.JWTClaims) error {
		if claims.Subject == "" {
			return respondError(http.StatusForbidden, TokenIsNotUserToken,
				"access forbidden: "+ErrorCodeMapping[TokenIsNotUserToken])
		}

		return nil
	}
}

// WithPermission filters request with valid permission only
func WithPermission(permission *iam.Permission) FilterOption {
	return func(req *restful.Request, iamClient iam.Client, claims *iam.JWTClaims) error {
		requiredPermissionResources := make(map[string]string)
		requiredPermissionResources["{namespace}"] = req.PathParameter("namespace")
		requiredPermissionResources["{userId}"] = req.PathParameter("userId")

		valid, err := iamClient.ValidatePermission(claims, *permission, requiredPermissionResources)
		if err != nil {
			return respondError(http.StatusInternalServerError, InternalServerError,
				"unable to validate permission: "+err.Error())
		}

		if !valid {
			return respondError(http.StatusForbidden, InsufficientPermissions,
				"access forbidden: "+ErrorCodeMapping[InsufficientPermissions])
		}

		return nil
	}
}

// WithRole filters request with valid role only
func WithRole(role string) FilterOption {
	return func(req *restful.Request, iamClient iam.Client, claims *iam.JWTClaims) error {
		valid, err := iamClient.ValidateRole(role, claims)
		if err != nil {
			return respondError(http.StatusInternalServerError, EIDWithRoleUnableValidateRole,
				"unable to validate role: "+err.Error())
		}

		if !valid {
			return respondError(http.StatusForbidden, EIDWithRoleInsufficientPermission,
				"access forbidden: insufficient permission")
		}

		return nil
	}
}

// WithVerifiedEmail filters request from a user with verified email address only
func WithVerifiedEmail() FilterOption {
	return func(req *restful.Request, iamClient iam.Client, claims *iam.JWTClaims) error {
		verified, err := iamClient.UserEmailVerificationStatus(claims)
		if err != nil {
			return respondError(http.StatusInternalServerError, EIDWithVerifiedEmailUnableValidateEmailStatus,
				"unable to validate email status: "+err.Error())
		}

		if !verified {
			return respondError(http.StatusForbidden, EIDWithVerifiedEmailInsufficientPermission,
				"access forbidden: insufficient permission")
		}

		return nil
	}
}

// WithValidAudience filters request from a user with verified audience
func WithValidAudience() FilterOption {
	return func(req *restful.Request, iamClient iam.Client, claims *iam.JWTClaims) error {
		err := iamClient.ValidateAudience(claims)
		if err != nil {
			return respondError(http.StatusForbidden, InvalidAudience,
				"access forbidden: "+ErrorCodeMapping[InvalidAudience])
		}

		return nil
	}
}

// WithValidScope filters request from a user with verified scope
func WithValidScope(scope string) FilterOption {
	return func(req *restful.Request, iamClient iam.Client, claims *iam.JWTClaims) error {
		err := iamClient.ValidateScope(claims, scope)
		if err != nil {
			return respondError(http.StatusForbidden, InsufficientScope,
				"access forbidden: "+ErrorCodeMapping[InsufficientScope])
		}

		return nil
	}
}

// WithMatchedSubdomain filters request to a subdomain to match it with namespace in user's token
func WithMatchedSubdomain(excludedNamespaces []string) FilterOption {
	return func(req *restful.Request, iamClient iam.Client, claims *iam.JWTClaims) error {
		part := strings.Split(getHost(req.Request), ".")
		if len(part) < 3 {
			// url with subdomain should have at least 3 part, e.g. foo.example.com, otherwise we should not check it
			return nil
		}

		for _, excludedNS := range excludedNamespaces {
			if strings.ToLower(excludedNS) == strings.ToLower(claims.Namespace) {
				return nil
			}
		}

		if strings.ToLower(claims.Namespace) == strings.ToLower(part[0]) {
			return nil
		}

		return respondError(http.StatusNotFound, SubdomainMismatch,
			"data not found: "+ErrorCodeMapping[SubdomainMismatch])
	}
}

func getHost(req *http.Request) string {
	if !req.URL.IsAbs() {
		host := req.Host
		if i := strings.Index(host, ":"); i != -1 {
			host = host[:i]
		}
		return host
	}
	return req.URL.Host
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

// validateRefererHeader is used validate the referer header against client's redirectURIs.
// we're not using Origin header since it will null for GET request.
func (filter *Filter) validateRefererHeader(request *restful.Request, claims *iam.JWTClaims) bool {
	clientInfo, err := filter.iamClient.GetClientInformation(claims.Namespace, claims.ClientID)
	if err != nil {
		logrus.Errorf("validate referer header error: %v", err.Error())
		return false
	}
	if len(clientInfo.RedirectURI) == 0 {
		return true
	}

	referer := request.HeaderParameter(constant.Referer)
	if referer != "" {
		refererDomain := util.GetDomain(referer)
		clientRedirectURIs := strings.Split(clientInfo.RedirectURI, ",")
		for _, redirectURI := range clientRedirectURIs {
			if filter.options.AllowSubdomainMatchRefererHeaderValidation {
				if validateRefererWithoutSubdomain(referer, redirectURI) {
					return true
				}
			} else {
				redirectURIDomain := util.GetDomain(redirectURI)
				if filter.options.StrictRefererHeaderValidation {
					if refererDomain == redirectURIDomain && strings.HasPrefix(referer, redirectURI) {
						return true
					}
				} else {
					if refererDomain == redirectURIDomain {
						return true
					}
				}
			}
		}
	}

	logrus.Warnf("request has invalid referer header. referer header: %s. client redirect uri: %s",
		referer, clientInfo.RedirectURI)
	return false
}

func validateRefererWithoutSubdomain(refererHeader string, clientRedirectURI string) bool {
	refererURL, err := url.Parse(refererHeader)
	if err != nil {
		return false
	}

	clientRedirectURL, err := url.Parse(clientRedirectURI)
	if err != nil {
		return false
	}

	if refererURL.Scheme != clientRedirectURL.Scheme {
		return false
	}

	// remove the ".www"
	clientRedirectHost := strings.Replace(clientRedirectURL.Host, "www.", "", 1)

	if strings.HasSuffix(refererURL.Host, clientRedirectHost) {
		// check the character after the redirectUri string in referer string,
		// if contains [a-zA-Z] character, then it's not a valid domain
		// e.g.
		// redirectUri host: accelbyte.io
		// referer host: mygame.evilaccelbyte.io
		if len(refererURL.Host) > len(clientRedirectHost) {
			if refererURL.Host[len(refererURL.Host)-len(clientRedirectHost)-1] != '.' {
				return false
			}
		}
		return true
	}

	return false
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
