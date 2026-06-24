/*
 *     Copyright 2025 The CNAI Authors
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

package pb

import (
	"io"
	"sync"
	"testing"
)

func TestSetDisableProgressIsRaceFree(t *testing.T) {
	progress := NewProgressBar(io.Discard)
	defer progress.Stop()
	defer SetDisableProgress(false)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			SetDisableProgress(i%2 == 0)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			progress.Add("copy", "blob", 1, nil)
		}
	}()

	wg.Wait()
}
