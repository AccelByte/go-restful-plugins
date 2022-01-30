// Copyright 2022 AccelByte Inc
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

package log

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskSingleField(t *testing.T) {
	t.Parallel()

	inputAndExpected := [][]string{
		// Content-Type = application/json
		{
			"application/json",                 // content-type
			"{\"password\":\"mypassword123\"}", // input
			"{\"password\":\"******\"}",        // expected
		},
		{
			"application/json",
			"{\"username\":\"my username\",\"password\":\"mypassword123\"}",
			"{\"username\":\"my username\",\"password\":\"******\"}",
		},
		{
			"application/json",
			"{\"username\":\"my username\",\"password\":\"mypassword123\",\"displayName\":\"My Display Name\"}",
			"{\"username\":\"my username\",\"password\":\"******\",\"displayName\":\"My Display Name\"}",
		},
		{
			"application/json",
			"{\"displayName\":\"My Display Name\",\"secret\":{\"username\":\"my username\",\"password\":\"mypassword123\"}}",
			"{\"displayName\":\"My Display Name\",\"secret\":{\"username\":\"my username\",\"password\":\"******\"}}",
		},
		// Content-Type = application/x-www-form-urlencoded
		{
			"application/x-www-form-urlencoded",
			"password=mypassword",
			"password=******",
		},
		{
			"application/x-www-form-urlencoded",
			"username=my username&password=my password",
			"username=my username&password=******",
		},
		{
			"application/x-www-form-urlencoded",
			"username=my username&password=my password&displayName=My Display Name",
			"username=my username&password=******&displayName=My Display Name",
		},
		// Content-Type = plain/text
		{
			"plain/text",
			"{\"username\":\"my username\",\"password\":\"mypassword123\",\"displayName\":\"My Display Name\"}",
			"{\"username\":\"my username\",\"password\":\"******\",\"displayName\":\"My Display Name\"}",
		},
		{
			"plain/text",
			"username=my username&password=mypassword&displayName=My Display Name",
			"username=my username&password=******&displayName=My Display Name",
		},
	}

	for _, val := range inputAndExpected {
		assert.Equal(t, val[2], MaskFields(val[0], val[1], "password"))
	}
}

func TestMaskMultipleFields(t *testing.T) {
	t.Parallel()

	inputAndExpected := [][]string{
		// Content-Type = application/json
		{
			"application/json",                 // content-type
			"{\"password\":\"mypassword123\"}", // input
			"{\"password\":\"******\"}",        // expected
		},
		{
			"application/json",
			"{\"username\":\"my username\",\"password\":\"mypassword123\"}",
			"{\"username\":\"******\",\"password\":\"******\"}",
		},
		{
			"application/json",
			"{\"username\":\"my username\",\"password\":\"mypassword123\",\"displayName\":\"My Display Name\"}",
			"{\"username\":\"******\",\"password\":\"******\",\"displayName\":\"My Display Name\"}",
		},
		{
			"application/json",
			"{\"displayName\":\"My Display Name\",\"secret\":{\"username\":\"my username\",\"password\":\"mypassword123\"}}",
			"{\"displayName\":\"My Display Name\",\"secret\":{\"username\":\"******\",\"password\":\"******\"}}",
		},
		// Content-Type = application/x-www-form-urlencoded
		{
			"application/x-www-form-urlencoded",
			"password=mypassword",
			"password=******",
		},
		{
			"application/x-www-form-urlencoded",
			"username=my username&password=my password",
			"username=******&password=******",
		},
		{
			"application/x-www-form-urlencoded",
			"username=my username&password=my password&displayName=My Display Name",
			"username=******&password=******&displayName=My Display Name",
		},
		// Content-Type = plain/text
		{
			"plain/text",
			"{\"username\":\"my username\",\"password\":\"mypassword123\",\"displayName\":\"My Display Name\"}",
			"{\"username\":\"******\",\"password\":\"******\",\"displayName\":\"My Display Name\"}",
		},
		{
			"plain/text",
			"username=my username&password=mypassword&displayName=My Display Name",
			"username=******&password=******&displayName=My Display Name",
		},
	}

	for _, val := range inputAndExpected {
		assert.Equal(t, val[2], MaskFields(val[0], val[1], "username,password"))
	}
}

func TestMaskMultipleFields_ButOneFieldIsNotExist(t *testing.T) {
	t.Parallel()

	inputAndExpected := [][]string{
		// Content-Type = application/json
		{
			"application/json",                 // content-type
			"{\"password\":\"mypassword123\"}", // input
			"{\"password\":\"******\"}",        // expected
		},
		{
			"application/json",
			"{\"username\":\"my username\",\"password\":\"mypassword123\"}",
			"{\"username\":\"my username\",\"password\":\"******\"}",
		},
		// Content-Type = application/x-www-form-urlencoded
		{
			"application/x-www-form-urlencoded",
			"password=mypassword",
			"password=******",
		},
		{
			"application/x-www-form-urlencoded",
			"username=my username&password=my password",
			"username=my username&password=******",
		},
		// Content-Type = plain/text
		{
			"plain/text",
			"{\"username\":\"my username\",\"password\":\"mypassword123\",\"displayName\":\"My Display Name\"}",
			"{\"username\":\"my username\",\"password\":\"******\",\"displayName\":\"My Display Name\"}",
		},
		{
			"plain/text",
			"username=my username&password=mypassword&displayName=My Display Name",
			"username=my username&password=******&displayName=My Display Name",
		},
	}

	for _, val := range inputAndExpected {
		assert.Equal(t, val[2], MaskFields(val[0], val[1], "password,apiKey"))
	}
}

func TestMaskField_ButNoneFieldIsExist(t *testing.T) {
	t.Parallel()

	inputAndExpected := [][]string{
		// Content-Type = application/json
		{
			"application/json",                      // content-type
			"{\"displayName\":\"My Display Name\"}", // input
			"{\"displayName\":\"My Display Name\"}", // expected
		},
		// Content-Type = application/x-www-form-urlencoded
		{
			"application/x-www-form-urlencoded",
			"displayName=My Display Name",
			"displayName=My Display Name",
		},
		// Content-Type = plain/text
		{
			"plain/text",
			"{\"displayName\":\"My Display Name\"}",
			"{\"displayName\":\"My Display Name\"}",
		},
		{
			"plain/text",
			"displayName=My Display Name",
			"displayName=My Display Name",
		},
	}

	for _, val := range inputAndExpected {
		assert.Equal(t, val[2], MaskFields(val[0], val[1], "password,apiKey"))
	}
}

func TestMaskField_ConcurrentCall(t *testing.T) {
	t.Parallel()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		input := "{\"password\":\"mypassword123\"}"
		expected := "{\"password\":\"******\"}"
		i := 0
		for i < 1000 {
			assert.Equal(t, expected, MaskFields("application/json", input, "password"))
			i++
		}
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		input := "{\"token\":\"my token 123\"}"
		expected := "{\"token\":\"******\"}"

		i := 0
		for i < 1000 {
			assert.Equal(t, expected, MaskFields("application/json", input, "token"))
			i++
		}
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		input := "apikey=my api key"
		expected := "apikey=******"

		i := 0
		for i < 1000 {
			assert.Equal(t, expected, MaskFields("application/x-www-form-urlencoded", input, "apikey"))
			i++
		}
		wg.Done()
	}()

	wg.Wait()
}

func TestMaskSingleQueryParam(t *testing.T) {
	t.Parallel()

	inputAndExpected := [][]string{
		{
			"https://example.net", // input
			"https://example.net", // expected
		},
		{
			"https://example.net?password=mypassword123",
			"https://example.net?password=******",
		},
		{
			"https://example.net?username=myusername&password=mypassword123",
			"https://example.net?username=myusername&password=******",
		},
		{
			"https://example.net?username=my username&password=mypassword 123",
			"https://example.net?username=my username&password=******",
		},
		{
			"https://example.net?username=my username&password=mypassword 123&displayName=My Display Name",
			"https://example.net?username=my username&password=******&displayName=My Display Name",
		},
		{
			"127.0.0.1?password=mypassword123",
			"127.0.0.1?password=******",
		},
		{
			"https://subdomain.example.net?password=mypassword123",
			"https://subdomain.example.net?password=******",
		},
	}

	for _, val := range inputAndExpected {
		assert.Equal(t, val[1], MaskQueryParams(val[0], "password"))
	}
}

func TestMaskMultipleQueryParams(t *testing.T) {
	t.Parallel()

	inputAndExpected := [][]string{
		{
			"https://example.net?password=mypassword123", // input
			"https://example.net?password=******",        // expected
		},
		{
			"https://example.net?username=myusername&password=mypassword123",
			"https://example.net?username=******&password=******",
		},
		{
			"https://example.net?username=my username&password=mypassword 123",
			"https://example.net?username=******&password=******",
		},
		{
			"https://example.net?username=my username&password=mypassword 123&displayName=My Display Name",
			"https://example.net?username=******&password=******&displayName=My Display Name",
		},
	}

	for _, val := range inputAndExpected {
		assert.Equal(t, val[1], MaskQueryParams(val[0], "username,password"))
	}
}

func TestMaskQueryParamOfEncodedURLs(t *testing.T) {
	t.Parallel()

	inputAndExpected := [][]string{
		{
			"https://example.net?username=my%20username&password=mypassword%20123", // input
			"https://example.net?username=******&password=******",                  // expected
		},
		{
			"https://example.net?username=my%20username&password=mypassword%20123&displayName=My%20Display%20Name",
			"https://example.net?username=******&password=******&displayName=My%20Display%20Name",
		},
		{
			// https://example.net?openid.ns=http://specs.openid.net/auth/2.0&openid.mode=id_res&openid.identity=https://steamcommunity.com/openid/id/123456789&openid.signed=signed,op_endpoint,claimed_id,identity&openid.sig=dGhpc19pc19zaWc=
			"https://example.net?openid.ns=http%3A%2F%2Fspecs.openid.net%2Fauth%2F2.0&openid.mode=id_res&openid.identity=https%3A%2F%2Fsteamcommunity.com%2Fopenid%2Fid%2F123456789&openid.signed=signed%2Cop_endpoint%2Cclaimed_id%2Cidentity&openid.sig=dGhpc19pc19zaWc%3D",
			"https://example.net?openid.ns=http%3A%2F%2Fspecs.openid.net%2Fauth%2F2.0&openid.mode=id_res&openid.identity=******&openid.signed=signed%2Cop_endpoint%2Cclaimed_id%2Cidentity&openid.sig=******",
		},
	}

	for _, val := range inputAndExpected {
		assert.Equal(t, val[1], MaskQueryParams(val[0], "username,password,openid.identity,openid.sig"))
	}
}

func TestMaskQueryParam_ConcurrentCall(t *testing.T) {
	t.Parallel()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		input := "https://example.net?password=mypassword123"
		expected := "https://example.net?password=******"
		i := 0
		for i < 1000 {
			assert.Equal(t, expected, MaskQueryParams(input, "password"))
			i++
		}
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		input := "https://example.net?username=username"
		expected := "https://example.net?username=******"
		i := 0
		for i < 1000 {
			assert.Equal(t, expected, MaskQueryParams(input, "username"))
			i++
		}
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		input := "https://example.net?apiKey=apiKey"
		expected := "https://example.net?apiKey=******"
		i := 0
		for i < 1000 {
			assert.Equal(t, expected, MaskQueryParams(input, "apiKey"))
			i++
		}
		wg.Done()
	}()

	wg.Wait()
}
