/*
 * Copyright 2018 AccelByte Inc
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

package iam

import (
	"errors"
	"net/http"
	"strings"

	"github.com/AccelByte/iam-go-sdk"
	"github.com/emicklei/go-restful"
	"github.com/sirupsen/logrus"
)

// ClaimsAttribute is the key for JWT claims stored in the request
const ClaimsAttribute = "JWTClaims"

// FilterOption extends the basic auth filter functionality
type FilterOption func(req *restful.Request, iamClient iam.Client, claims *iam.JWTClaims) error

// Filter handles auth using filter
type Filter struct {
	iamClient iam.Client
}

// NewFilter creates new Filter instance
func NewFilter(client iam.Client) *Filter {
	return &Filter{iamClient: client}
}

// Auth returns a filter that filters request with valid access token in auth header
// The token's claims will be passed in the request.attributes["JWTClaims"] = *iam.JWTClaims{}
// This filter is expandable through FilterOption parameter
// Example:
// iam.Auth(
// 		WithValidUser(),
//		WithPermission("ADMIN"),
// )
func (filter *Filter) Auth(opts ...FilterOption) restful.FilterFunction {
	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		token, err := parseAccessToken(req)
		if err != nil {
			logrus.Warn("unauthorized access: ", err)
			logErr(resp.WriteErrorString(http.StatusUnauthorized, "unauthorized access"))
			return
		}

		claims, err := filter.iamClient.ValidateAndParseClaims(token)
		if err != nil {
			logrus.Warn("unauthorized access: ", err)
			logErr(resp.WriteErrorString(http.StatusUnauthorized, "unauthorized access"))
			return
		}

		for _, opt := range opts {
			if err = opt(req, filter.iamClient, claims); err != nil {
				if svcErr, ok := err.(restful.ServiceError); ok {
					logrus.Warn(svcErr.Message)
					logErr(resp.WriteErrorString(svcErr.Code, svcErr.Message))
					return
				}
				logrus.Warn(err)
				logErr(resp.WriteErrorString(http.StatusUnauthorized, err.Error()))
				return
			}
		}

		req.SetAttribute(ClaimsAttribute, claims)

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
			return restful.NewError(http.StatusForbidden, "access forbidden: non user access token")
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
			return restful.NewError(http.StatusInternalServerError, "unable to validate permission: "+err.Error())
		}
		if !valid {
			return restful.NewError(http.StatusForbidden, "access forbidden: insufficient permission")
		}
		return nil
	}
}

// WithRole filters request with valid role only
func WithRole(role string) FilterOption {
	return func(req *restful.Request, iamClient iam.Client, claims *iam.JWTClaims) error {
		valid, err := iamClient.ValidateRole(role, claims)
		if err != nil {
			return restful.NewError(http.StatusInternalServerError, "unable to validate role: "+err.Error())
		}
		if !valid {
			return restful.NewError(http.StatusForbidden, "access forbidden: insufficient permission")
		}
		return nil
	}
}

// WithVerifiedEmail filters request from a user with verified email address only
func WithVerifiedEmail() FilterOption {
	return func(req *restful.Request, iamClient iam.Client, claims *iam.JWTClaims) error {
		verified, err := iamClient.UserEmailVerificationStatus(claims)
		if err != nil {
			return restful.NewError(http.StatusInternalServerError, "unable to validate email status: "+err.Error())
		}
		if !verified {
			return restful.NewError(http.StatusForbidden, "access forbidden: insufficient permission")
		}
		return nil
	}
}

func parseAccessToken(request *restful.Request) (string, error) {
	authorization := request.HeaderParameter("Authorization")
	if authorization == "" {
		return "", errors.New("unable to get Authorization header")
	}

	tokenSplit := strings.Split(authorization, " ")
	if len(tokenSplit) != 2 || tokenSplit[0] != "Bearer" {
		return "", errors.New("incorrect token")
	}
	token := tokenSplit[1]

	return token, nil
}

func logErr(err error) {
	logrus.Error(err)
}
