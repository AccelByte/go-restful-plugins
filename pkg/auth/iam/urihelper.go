// Copyright (c) 2021 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package iam

import "strings"

// getDomain is used to get domain (scheme+host) from the specified URI
func getDomain(uri string) string {
	afterDoubleSlashIndex := strings.Index(uri, "//")
	if afterDoubleSlashIndex == -1 {
		afterDoubleSlashIndex = 0
	} else {
		afterDoubleSlashIndex += 2
	}

	pathIndex := afterDoubleSlashIndex + strings.Index(uri[afterDoubleSlashIndex:], "/")
	if pathIndex > afterDoubleSlashIndex {
		return uri[:pathIndex]
	}
	return uri
}