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
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKnownBug_DisableProgress_DataRace(t *testing.T) {
	// This test documents the data race in SetDisableProgress/Add.
	// Run with: go test -race ./internal/pb/ -run TestKnownBug_DisableProgress_DataRace
	//
	// Known bug: global disableProgress bool has no atomic protection.
	// See: https://github.com/modelpack/modctl/issues/493
	//
	// When the bug is fixed (atomic.Bool), this test will still pass
	// AND the -race detector will stop reporting the race.
	// At that point, remove the KnownBug prefix.

	var wg sync.WaitGroup
	pb := NewProgressBar(io.Discard)
	pb.Start()
	defer pb.Stop()

	// Concurrent SetDisableProgress.
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				SetDisableProgress(j%2 == 0)
			}
		}()
	}

	// Concurrent Add.
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				reader := strings.NewReader("test data")
				pb.Add("test", "bar-"+string(rune('a'+id)), 9, reader)
			}
		}(i)
	}

	wg.Wait()
	// If we get here without panic, the test passes.
	// The real detection is via -race flag.
}

func TestIntegration_ProgressBar_ConcurrentUpdates(t *testing.T) {
	pb := NewProgressBar(io.Discard)
	pb.Start()
	defer pb.Stop()

	// Disable progress to avoid mpb rendering issues in test.
	SetDisableProgress(true)
	defer SetDisableProgress(false)

	var wg sync.WaitGroup

	// Concurrent Add + Complete + Abort from multiple goroutines.
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			name := string(rune('a' + id))
			reader := strings.NewReader("data")
			pb.Add("prompt", name, 4, reader)
			if id%3 == 0 {
				pb.Complete(name, "done")
			} else if id%3 == 1 {
				pb.Abort(name, assert.AnError)
			}
		}(i)
	}

	wg.Wait()
	// No panic = success.
}
