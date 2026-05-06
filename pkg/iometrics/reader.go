/*
 *     Copyright 2024 The ModelPack Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package iometrics

import (
	"io"
	"sync/atomic"
	"time"
)

// countingReader wraps an io.Reader, atomically incrementing shared byte
// and nanosecond counters on every Read() call. Multiple countingReader
// instances can share the same counters for cross-goroutine aggregation.
type countingReader struct {
	reader io.Reader
	bytes  *atomic.Int64
	nanos  *atomic.Int64
}

func (r *countingReader) Read(p []byte) (int, error) {
	start := time.Now()
	n, err := r.reader.Read(p)
	r.nanos.Add(int64(time.Since(start)))
	if n > 0 {
		r.bytes.Add(int64(n))
	}
	return n, err
}
