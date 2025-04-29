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

func TestMaskPIIFieldsInQuery_MaskMultipleFields(t *testing.T) {
	t.Parallel()

	inputAndExpected := [][]string{
		// Content-Type = application/json
		{
			"application/json", // content-type
			"{\"username\":\"username12\", \"emails\":[\"test1@accelbyte.net\",\"test2@accelbyte.net\"]}",       // input
			"{\"username\":\"username****\", \"emails\":[\"tes****@accelbyte.net\",\"tes****@accelbyte.net\"]}", // expected
		},
		{
			"application/json",
			"{\"username\":\"username12\",\"password\":\"mypassword123\"}",
			"{\"username\":\"username****\",\"password\":\"mypassword123\"}",
		},
		{
			"application/json",
			"{\"username\":\"username12\",\"emailAddress\":\"test@accelbyte.net\",\"displayName\":\"My Display Name\"}",
			"{\"username\":\"username****\",\"emailAddress\":\"te****@accelbyte.net\",\"displayName\":\"My Display Name\"}",
		},
		{
			"application/json",
			"{\"displayName\":\"My Display Name\",\"data\":{\"username\":\"username12\",\"emailAddress\":\"test@accelbyte.net\"}}",
			"{\"displayName\":\"My Display Name\",\"data\":{\"username\":\"username****\",\"emailAddress\":\"te****@accelbyte.net\"}}",
		},
		{
			"application/json",
			"{\"username\":\"us\",\"emailAddress\":\"te@accelbyte.net\",\"displayName\":\"My Display Name\"}",
			"{\"username\":\"u****\",\"emailAddress\":\"t****@accelbyte.net\",\"displayName\":\"My Display Name\"}",
		},
		{
			"application/json",
			"{\"displayName\":\"My Display Name\",\"data\":{\"username\":\"use\",\"emailAddress\":\"tes@accelbyte.net\"}}",
			"{\"displayName\":\"My Display Name\",\"data\":{\"username\":\"u****\",\"emailAddress\":\"t****@accelbyte.net\"}}",
		},
		// Content-Type = application/x-www-form-urlencoded
		{
			"application/x-www-form-urlencoded",
			"username=username12",
			"username=username****",
		},
		{
			"application/x-www-form-urlencoded",
			"username=username12&password=testaccelbytenet",
			"username=username****&password=testaccelbytenet",
		},
		{
			"application/x-www-form-urlencoded",
			"username=username12&emailAddress=test@accelbyte.net&displayName=My Display Name",
			"username=username****&emailAddress=te****@accelbyte.net&displayName=My Display Name",
		},
		{
			"application/x-www-form-urlencoded",
			"username=use&password=testaccelbytenet&emailAddress=tes@accelbyte.net",
			"username=u****&password=testaccelbytenet&emailAddress=t****@accelbyte.net",
		},
		{
			"application/x-www-form-urlencoded",
			"username=us&emailAddress=te@accelbyte.net&displayName=My Display Name",
			"username=u****&emailAddress=t****@accelbyte.net&displayName=My Display Name",
		},
		// Content-Type = plain/text
		{
			"plain/text",
			"{\"username\":\"username12\",\"emailAddress\":\"test@accelbyte.net\",\"displayName\":\"My Display Name\", \"emails\":[\"test1@accelbyte.net\",\"test2@accelbyte.net\"]}",
			"{\"username\":\"username****\",\"emailAddress\":\"te****@accelbyte.net\",\"displayName\":\"My Display Name\", \"emails\":[\"tes****@accelbyte.net\",\"tes****@accelbyte.net\"]}",
		},
		{
			"plain/text",
			"{\"username\":\"us\",\"emailAddress\":\"te@accelbyte.net\",\"displayName\":\"My Display Name\"}",
			"{\"username\":\"u****\",\"emailAddress\":\"t****@accelbyte.net\",\"displayName\":\"My Display Name\"}",
		},
		{
			"plain/text",
			"{\"username\":\"use\",\"emailAddress\":\"tes@accelbyte.net\",\"displayName\":\"My Display Name\"}",
			"{\"username\":\"u****\",\"emailAddress\":\"t****@accelbyte.net\",\"displayName\":\"My Display Name\"}",
		},
		{
			"plain/text",
			"username=username12&emailAddress=test@accelbyte.net&displayName=My Display Name",
			"username=username****&emailAddress=te****@accelbyte.net&displayName=My Display Name",
		},
		{
			"plain/text",
			"username=use&emailAddress=tes@accelbyte.net&displayName=My Display Name",
			"username=u****&emailAddress=t****@accelbyte.net&displayName=My Display Name",
		},
		{
			"plain/text",
			"username=us&emailAddress=te@accelbyte.net&displayName=My Display Name",
			"username=u****&emailAddress=t****@accelbyte.net&displayName=My Display Name",
		},
	}

	for _, val := range inputAndExpected {
		assert.Equal(t, val[2], MaskPIIFields(val[0], val[1], "username,emailAddress,emails"))
	}
}

func TestMaskPIIFieldsInQuery_MaskSingleFieldUsername(t *testing.T) {
	t.Parallel()

	inputAndExpected := [][]string{
		// Content-Type = application/json
		{
			"application/json",                // content-type
			"{\"username\":\"username12\"}",   // input
			"{\"username\":\"username****\"}", // expected
		},
		{
			"application/json",
			"{\"username\":\"username12\",\"password\":\"mypassword123\"}",
			"{\"username\":\"username****\",\"password\":\"mypassword123\"}",
		},
		{
			"application/json",
			"{\"username\":\"username12\",\"emailAddress\":\"test@accelbyte.net\",\"displayName\":\"My Display Name\"}",
			"{\"username\":\"username****\",\"emailAddress\":\"test@accelbyte.net\",\"displayName\":\"My Display Name\"}",
		},
		{
			"application/json",
			"{\"displayName\":\"My Display Name\",\"data\":{\"username\":\"username12\",\"emailAddress\":\"test@accelbyte.net\"}}",
			"{\"displayName\":\"My Display Name\",\"data\":{\"username\":\"username****\",\"emailAddress\":\"test@accelbyte.net\"}}",
		},
		{
			"application/json",
			"{\"username\":\"us\",\"emailAddress\":\"te@accelbyte.net\",\"displayName\":\"My Display Name\"}",
			"{\"username\":\"u****\",\"emailAddress\":\"te@accelbyte.net\",\"displayName\":\"My Display Name\"}",
		},
		{
			"application/json",
			"{\"displayName\":\"My Display Name\",\"data\":{\"username\":\"use\",\"emailAddress\":\"tes@accelbyte.net\"}}",
			"{\"displayName\":\"My Display Name\",\"data\":{\"username\":\"u****\",\"emailAddress\":\"tes@accelbyte.net\"}}",
		},
		// Content-Type = application/x-www-form-urlencoded
		{
			"application/x-www-form-urlencoded",
			"username=username12",
			"username=username****",
		},
		{
			"application/x-www-form-urlencoded",
			"username=username12&password=testaccelbytenet",
			"username=username****&password=testaccelbytenet",
		},
		{
			"application/x-www-form-urlencoded",
			"username=username12&emailAddress=test@accelbyte.net&displayName=My Display Name",
			"username=username****&emailAddress=test@accelbyte.net&displayName=My Display Name",
		},
		{
			"application/x-www-form-urlencoded",
			"username=use&password=testaccelbytenet&emailAddress=tes@accelbyte.net",
			"username=u****&password=testaccelbytenet&emailAddress=tes@accelbyte.net",
		},
		{
			"application/x-www-form-urlencoded",
			"username=us&emailAddress=te@accelbyte.net&displayName=My Display Name",
			"username=u****&emailAddress=te@accelbyte.net&displayName=My Display Name",
		},
		// Content-Type = plain/text
		{
			"plain/text",
			"{\"username\":\"username12\",\"emailAddress\":\"test@accelbyte.net\",\"displayName\":\"My Display Name\"}",
			"{\"username\":\"username****\",\"emailAddress\":\"test@accelbyte.net\",\"displayName\":\"My Display Name\"}",
		},
		{
			"plain/text",
			"{\"username\":\"us\",\"emailAddress\":\"te@accelbyte.net\",\"displayName\":\"My Display Name\"}",
			"{\"username\":\"u****\",\"emailAddress\":\"te@accelbyte.net\",\"displayName\":\"My Display Name\"}",
		},
		{
			"plain/text",
			"{\"username\":\"use\",\"emailAddress\":\"tes@accelbyte.net\",\"displayName\":\"My Display Name\"}",
			"{\"username\":\"u****\",\"emailAddress\":\"tes@accelbyte.net\",\"displayName\":\"My Display Name\"}",
		},
		{
			"plain/text",
			"username=username12&emailAddress=test@accelbyte.net&displayName=My Display Name",
			"username=username****&emailAddress=test@accelbyte.net&displayName=My Display Name",
		},
		{
			"plain/text",
			"username=use&emailAddress=tes@accelbyte.net&displayName=My Display Name",
			"username=u****&emailAddress=tes@accelbyte.net&displayName=My Display Name",
		},
		{
			"plain/text",
			"username=us&emailAddress=te@accelbyte.net&displayName=My Display Name",
			"username=u****&emailAddress=te@accelbyte.net&displayName=My Display Name",
		},
	}

	for _, val := range inputAndExpected {
		assert.Equal(t, val[2], MaskPIIFields(val[0], val[1], "username"))
	}
}

func TestMaskPIIFieldsInQuery_MaskSingleFieldEmailAddress(t *testing.T) {
	t.Parallel()

	inputAndExpected := [][]string{
		// Content-Type = application/json
		{
			"application/json",                            // content-type
			"{\"emailAddress\":\"test@accelbyte.net\"}",   // input
			"{\"emailAddress\":\"te****@accelbyte.net\"}", // expected
		},
		{
			"application/json",
			"{\"emailAddress\":\"test@accelbyte.net\",\"password\":\"mypassword123\"}",
			"{\"emailAddress\":\"te****@accelbyte.net\",\"password\":\"mypassword123\"}",
		},
		{
			"application/json",
			"{\"username\":\"username12\",\"emailAddress\":\"test@accelbyte.net\",\"displayName\":\"My Display Name\"}",
			"{\"username\":\"username12\",\"emailAddress\":\"te****@accelbyte.net\",\"displayName\":\"My Display Name\"}",
		},
		{
			"application/json",
			"{\"displayName\":\"My Display Name\",\"data\":{\"username\":\"username12\",\"emailAddress\":\"test@accelbyte.net\"}}",
			"{\"displayName\":\"My Display Name\",\"data\":{\"username\":\"username12\",\"emailAddress\":\"te****@accelbyte.net\"}}",
		},
		{
			"application/json",
			"{\"username\":\"us\",\"emailAddress\":\"te@accelbyte.net\",\"displayName\":\"My Display Name\"}",
			"{\"username\":\"us\",\"emailAddress\":\"t****@accelbyte.net\",\"displayName\":\"My Display Name\"}",
		},
		{
			"application/json",
			"{\"displayName\":\"My Display Name\",\"data\":{\"username\":\"use\",\"emailAddress\":\"tes@accelbyte.net\"}}",
			"{\"displayName\":\"My Display Name\",\"data\":{\"username\":\"use\",\"emailAddress\":\"t****@accelbyte.net\"}}",
		},
		// Content-Type = application/x-www-form-urlencoded
		{
			"application/x-www-form-urlencoded",
			"emailAddress=test@accelbyte.net",
			"emailAddress=te****@accelbyte.net",
		},
		{
			"application/x-www-form-urlencoded",
			"emailAddress=test@accelbyte.net&password=testaccelbytenet",
			"emailAddress=te****@accelbyte.net&password=testaccelbytenet",
		},
		{
			"application/x-www-form-urlencoded",
			"username=username12&emailAddress=test@accelbyte.net&displayName=My Display Name",
			"username=username12&emailAddress=te****@accelbyte.net&displayName=My Display Name",
		},
		{
			"application/x-www-form-urlencoded",
			"username=use&password=testaccelbytenet&emailAddress=tes@accelbyte.net",
			"username=use&password=testaccelbytenet&emailAddress=t****@accelbyte.net",
		},
		{
			"application/x-www-form-urlencoded",
			"username=us&emailAddress=te@accelbyte.net&displayName=My Display Name",
			"username=us&emailAddress=t****@accelbyte.net&displayName=My Display Name",
		},
		// Content-Type = plain/text
		{
			"plain/text",
			"{\"username\":\"username12\",\"emailAddress\":\"test@accelbyte.net\",\"displayName\":\"My Display Name\"}",
			"{\"username\":\"username12\",\"emailAddress\":\"te****@accelbyte.net\",\"displayName\":\"My Display Name\"}",
		},
		{
			"plain/text",
			"{\"username\":\"us\",\"emailAddress\":\"te@accelbyte.net\",\"displayName\":\"My Display Name\"}",
			"{\"username\":\"us\",\"emailAddress\":\"t****@accelbyte.net\",\"displayName\":\"My Display Name\"}",
		},
		{
			"plain/text",
			"{\"username\":\"use\",\"emailAddress\":\"tes@accelbyte.net\",\"displayName\":\"My Display Name\"}",
			"{\"username\":\"use\",\"emailAddress\":\"t****@accelbyte.net\",\"displayName\":\"My Display Name\"}",
		},
		{
			"plain/text",
			"username=username12&emailAddress=test@accelbyte.net&displayName=My Display Name",
			"username=username12&emailAddress=te****@accelbyte.net&displayName=My Display Name",
		},
		{
			"plain/text",
			"username=use&emailAddress=tes@accelbyte.net&displayName=My Display Name",
			"username=use&emailAddress=t****@accelbyte.net&displayName=My Display Name",
		},
		{
			"plain/text",
			"username=us&emailAddress=te@accelbyte.net&displayName=My Display Name",
			"username=us&emailAddress=t****@accelbyte.net&displayName=My Display Name",
		},
	}

	for _, val := range inputAndExpected {
		assert.Equal(t, val[2], MaskPIIFields(val[0], val[1], "emailAddress"))
	}
}

func TestMaskMultiplePIIFields_ButOneFieldDoesNotExist(t *testing.T) {
	t.Parallel()

	inputAndExpected := [][]string{
		// Content-Type = application/json
		{
			"application/json",                            // content-type
			"{\"emailAddress\":\"test@accelbyte.net\"}",   // input
			"{\"emailAddress\":\"te****@accelbyte.net\"}", // expected
		},
		{
			"application/json",
			"{\"emailAddress\":\"test@accelbyte.net\",\"password\":\"mypassword123\"}",
			"{\"emailAddress\":\"te****@accelbyte.net\",\"password\":\"mypassword123\"}",
		},
		// Content-Type = application/x-www-form-urlencoded
		{
			"application/x-www-form-urlencoded",
			"username=username12",
			"username=username****",
		},
		{
			"application/x-www-form-urlencoded",
			"username=use&password=my password",
			"username=u****&password=my password",
		},
		// Content-Type = plain/text
		{
			"plain/text",
			"{\"username\":\"username12\",\"emailAddress\":\"test@accelbyte.net\",\"displayName\":\"My Display Name\"}",
			"{\"username\":\"username****\",\"emailAddress\":\"te****@accelbyte.net\",\"displayName\":\"My Display Name\"}",
		},
		{
			"plain/text",
			"username=username12&emailAddress=test@accelbyte.net&displayName=My Display Name",
			"username=username****&emailAddress=te****@accelbyte.net&displayName=My Display Name",
		},
	}

	for _, val := range inputAndExpected {
		assert.Equal(t, val[2], MaskPIIFields(val[0], val[1], "emailAddress,username,apiKey"))
	}
}

func TestMaskPIIFields_ConcurrentCall(t *testing.T) {
	t.Parallel()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		input := "{\"username\":\"username12\", \"emails\":[\"test1@accelbyte.net\",\"test2@accelbyte.net\"]}"
		expected := "{\"username\":\"username****\", \"emails\":[\"tes****@accelbyte.net\",\"tes****@accelbyte.net\"]}"
		i := 0
		for i < 1000 {
			assert.Equal(t, expected, MaskPIIFields("application/json", input, "username,emails"))
			i++
		}
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		input := "{\"emailAddress\":\"test@accelbyte.net\"}"
		expected := "{\"emailAddress\":\"te****@accelbyte.net\"}"

		i := 0
		for i < 1000 {
			assert.Equal(t, expected, MaskPIIFields("application/json", input, "emailAddress"))
			i++
		}
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		input := "username=use&emailAddress=tes@accelbyte.net&displayName=My Display Name"
		expected := "username=u****&emailAddress=t****@accelbyte.net&displayName=My Display Name"

		i := 0
		for i < 1000 {
			assert.Equal(t, expected, MaskPIIFields("application/x-www-form-urlencoded", input, "username,emailAddress"))
			i++
		}
		wg.Done()
	}()

	wg.Wait()
}

func TestMaskMultiplePIIQueryParams(t *testing.T) {
	t.Parallel()

	inputAndExpected := [][]string{
		{
			"https://example.net?username=username12",   // input
			"https://example.net?username=username****", // expected
		},
		{
			"https://example.net?username=username12&password=mypassword123&emailAddress=test@accelbyte.net",
			"https://example.net?username=username****&password=mypassword123&emailAddress=te****@accelbyte.net",
		},
		{
			"https://example.net?username=username 12&password=mypassword123&emailAddress=te@accelbyte.net",
			"https://example.net?username=username ****&password=mypassword123&emailAddress=t****@accelbyte.net",
		},
		{
			"https://example.net?username=user name12&password=mypassword 123&displayName=My Display Name",
			"https://example.net?username=user name****&password=mypassword 123&displayName=My Display Name",
		},
		{
			"https://example.net?loginIds=test1@accelbyte.net,test2@accelbyte.net&password=mypassword 123&displayName=My Display Name",
			"https://example.net?loginIds=tes****@accelbyte.net,tes****@accelbyte.net&password=mypassword 123&displayName=My Display Name",
		},
	}

	for _, val := range inputAndExpected {
		assert.Equal(t, val[1], MaskPIIQueryParams(val[0], "username,emailAddress,loginIds"))
	}
}

func TestMaskPIIQueryParamOfEncodedURLs(t *testing.T) {
	t.Parallel()

	inputAndExpected := [][]string{
		{
			"https://example.net?username=my%20username&emailAddress=test%40accelbyte.net", // input
			"https://example.net?username=my userna****&emailAddress=te****@accelbyte.net", // expected
		},
		{
			"https://example.net?username=username12&emailAddress=test%40accelbyte.net&displayName=My%20Display%20Name",
			"https://example.net?username=username****&emailAddress=te****@accelbyte.net&displayName=My Display Name",
		},
		{
			"https://example.net?openid.ns=http%3A%2F%2Fspecs.openid.net%2Fauth%2F2.0&openid.mode=id_res&openid.identity=https%3A%2F%2Fsteamcommunity.com%2Fopenid%2Fid%2F123456789&openid.signed=signed%2Cop_endpoint%2Cclaimed_id%2Cidentity&openid.sig=dGhpc19pc19zaWc%3D",
			"https://example.net?openid.ns=http://specs.openid.net/auth/2.0&openid.mode=id_res&openid.identity=https://steamcommunity.com/openid/id/123456789&openid.signed=signed,op_endpoint,claimed_id,identity&openid.sig=dGhpc19pc19zaWc=",
		},
		{
			"https://example.net?loginIds=test1%40accelbyte.net,test2%40accelbyte.net&displayName=My%20Display%20Name",
			"https://example.net?loginIds=tes****@accelbyte.net,tes****@accelbyte.net&displayName=My Display Name",
		},
	}

	for _, val := range inputAndExpected {
		assert.Equal(t, val[1], MaskPIIQueryParams(val[0], "username,emailAddress,loginIds"))
	}
}

func TestMaskPIIQueryParam_ConcurrentCall(t *testing.T) {
	t.Parallel()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		input := "https://example.net?emailAddress=test@accelbyte.net"
		expected := "https://example.net?emailAddress=te****@accelbyte.net"
		i := 0
		for i < 1000 {
			assert.Equal(t, expected, MaskPIIQueryParams(input, "emailAddress"))
			i++
		}
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		input := "https://example.net?emailAddress=test%40accelbyte.net"
		expected := "https://example.net?emailAddress=te****@accelbyte.net"
		i := 0
		for i < 1000 {
			assert.Equal(t, expected, MaskPIIQueryParams(input, "emailAddress"))
			i++
		}
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		input := "https://example.net?key=12&username=username12"
		expected := "https://example.net?key=12&username=username****"
		i := 0
		for i < 1000 {
			assert.Equal(t, expected, MaskPIIQueryParams(input, "username"))
			i++
		}
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		input := "https://example.net?username=username12&key=12"
		expected := "https://example.net?username=username****&key=12"
		i := 0
		for i < 1000 {
			assert.Equal(t, expected, MaskPIIQueryParams(input, "username"))
			i++
		}
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		input := "https://example.net?loginIds=test1@accelbyte.net,test2@accelbyte.net&key=12"
		expected := "https://example.net?loginIds=tes****@accelbyte.net,tes****@accelbyte.net&key=12"
		i := 0
		for i < 1000 {
			assert.Equal(t, expected, MaskPIIQueryParams(input, "loginIds"))
			i++
		}
		wg.Done()
	}()

	wg.Wait()
}
