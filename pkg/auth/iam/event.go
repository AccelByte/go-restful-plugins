// Copyright (c) 2019 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package iam

const (
	EIDWithValidUserNonUserAccessToken            = 1154001
	EIDWithPermissionUnableValidatePermission     = 1155001
	EIDWithPermissionInsufficientPermission       = 1154002
	EIDWithRoleUnableValidateRole                 = 1155002
	EIDWithRoleInsufficientPermission             = 1154003
	EIDWithVerifiedEmailUnableValidateEmailStatus = 1155003
	EIDWithVerifiedEmailInsufficientPermission    = 1154004
	EIDAccessDenied                               = 1154005
	EIDInsufficientScope                          = 1154006
	UnableToMarshalErrorResponse                  = 1155004
)

const (
	// Global Error Codes
	InternalServerError         = 20000
	UnauthorizedAccess          = 20001
	ValidationError             = 20002
	ForbiddenAccess             = 20003
	TooManyRequests             = 20007
	UserNotFound                = 20008
	InsufficientPermissions     = 20013
	InvalidAudience             = 20014
	InsufficientScope           = 20015
	UnableToParseRequestBody    = 20019
	InvalidPaginationParameters = 20021
	TokenIsNotUserToken         = 20022
)

var ErrorCodeMapping = map[int]string{
	// Global Error Codes
	InternalServerError:         "internal server error",
	UnauthorizedAccess:          "unauthorized access",
	ValidationError:             "validation error",
	ForbiddenAccess:             "forbidden access",
	TooManyRequests:             "too many requests",
	UserNotFound:                "user not found",
	InsufficientPermissions:     "insufficient permissions",
	InvalidAudience:             "invalid audience",
	InsufficientScope:           "insufficient scope",
	UnableToParseRequestBody:    "unable to parse request body",
	InvalidPaginationParameters: "invalid pagination parameter",
	TokenIsNotUserToken:         "token is not user token",
}
