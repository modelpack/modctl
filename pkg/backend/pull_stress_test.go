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

//go:build stress

package backend

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestStress_Pull_ManyLayers pulls a manifest with 50 blobs using concurrency
// 10 and asserts that the pull completes successfully within 60 seconds.
func TestStress_Pull_ManyLayers(t *testing.T) {
	const blobCount = 50

	f := newPullTestFixture(t, blobCount)
	defer f.mr.Close()

	f.cfg.Concurrency = 10

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	err := f.backend.Pull(ctx, f.target, f.cfg)
	require.NoError(t, err, "pull with %d layers should succeed", blobCount)

	// 50 layer blobs + 1 config blob = 51 PushBlob calls.
	f.store.AssertNumberOfCalls(t, "PushBlob", blobCount+1)
	f.store.AssertNumberOfCalls(t, "PushManifest", 1)
}

// TestStress_Pull_RepeatedCycles runs pull 100 times in a loop using a
// 2-blob fixture and asserts that the goroutine count stays stable (within a
// delta of 20), detecting goroutine leaks.
func TestStress_Pull_RepeatedCycles(t *testing.T) {
	const cycles = 100
	const goroutineDelta = 20

	f := newPullTestFixture(t, 2)
	defer f.mr.Close()

	goroutinesBefore := runtime.NumGoroutine()

	for i := 0; i < cycles; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err := f.backend.Pull(ctx, f.target, f.cfg)
		cancel()
		require.NoError(t, err, "pull cycle %d/%d should succeed", i+1, cycles)
	}

	// Give any background goroutines a moment to finish.
	runtime.Gosched()

	goroutinesAfter := runtime.NumGoroutine()
	leaked := goroutinesAfter - goroutinesBefore
	require.LessOrEqual(t, leaked, goroutineDelta,
		"goroutine count grew by %d after %d pull cycles (before=%d, after=%d); possible goroutine leak",
		leaked, cycles, goroutinesBefore, goroutinesAfter)
}
