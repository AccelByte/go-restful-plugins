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

package log

import (
	"bufio"
	"net"
	"net/http"
)

// ResponseWriterInterceptor is used to decorate http.ResponseWriter,
// so we can intercept the Write process
type ResponseWriterInterceptor struct {
	http.ResponseWriter

	// data is a copy of the response body, collected across all writes while captureBody is set. It is dropped,
	// and collection stops, once the body is larger than FullAccessLogMaxBodySize, since it is then logged as
	// bodyTooLarge. Whether the body is logged at all is decided afterwards, from its final Content-Type.
	data []byte
	// size is the number of response body bytes written.
	size        int
	captureBody bool
	// fromAccessLog is set on an interceptor that AccessLog created, so that a nested AccessLog reuses only those.
	fromAccessLog bool
}

func (w *ResponseWriterInterceptor) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	// Some writers report a byte count outside 0..len(b) with the error, e.g. go-restful's CompressingResponseWriter
	// returns -1 once it is closed. The count is clamped only for the access log; the caller gets the wrapped
	// writer's result unchanged.
	written := min(max(n, 0), len(b))
	w.size += written

	if w.captureBody {
		if w.size > FullAccessLogMaxBodySize {
			w.captureBody = false
			w.data = nil
		} else {
			w.data = append(w.data, b[:written]...)
		}
	}

	return n, err
}

// Flush sends the data written so far to the client, when a writer underneath can. It makes the interceptor an
// http.Flusher, so streaming responses, such as server-sent events, still flush behind AccessLog.
func (w *ResponseWriterInterceptor) Flush() {
	_ = w.FlushError()
}

// FlushError is Flush that reports the error, for http.ResponseController. It is passed down through the wrapped
// writers by http.ResponseController, and is http.ErrNotSupported when none of them can flush.
func (w *ResponseWriterInterceptor) FlushError() error {
	return http.NewResponseController(w.ResponseWriter).Flush()
}

// Hijack hands the connection over to the handler, e.g. for a WebSocket upgrade, when a writer underneath can. It
// makes the interceptor an http.Hijacker, is passed down through the wrapped writers, and is http.ErrNotSupported
// when none of them can hijack. Bytes written to a hijacked connection don't reach the access log.
func (w *ResponseWriterInterceptor) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return http.NewResponseController(w.ResponseWriter).Hijack()
}

// Unwrap returns the wrapped writer, so that http.ResponseController can reach its other features.
func (w *ResponseWriterInterceptor) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
