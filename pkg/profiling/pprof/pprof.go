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

package pprof

import (
	"net/http"
	netProf "net/http/pprof"
	"strings"
)

// Route registers pprof routes based on the basepath
func Route(basePath string) {
	http.HandleFunc(basePath+"/debug/pprof/", index(basePath))
	http.HandleFunc(basePath+"/debug/pprof/cmdline", netProf.Cmdline)
	http.HandleFunc(basePath+"/debug/pprof/profile", netProf.Profile)
	http.HandleFunc(basePath+"/debug/pprof/symbol", netProf.Symbol)
	http.HandleFunc(basePath+"/debug/pprof/trace", netProf.Trace)
}

func index(basePath string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = strings.TrimPrefix(r.URL.Path, basePath)
		netProf.Index(w, r)
	}
}
