//go:build stress

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

	// Every Pull creates its own http.Transport and leaves its idle
	// keep-alive connections open. The client readLoop/writeLoop and the
	// server conn.serve goroutines of those connections all live in this
	// process, so with keep-alives on the count grows by about 5 per cycle
	// without any real leak. Disable keep-alives to measure real leaks only.
	f.mr.SetKeepAlivesEnabled(false)

	goroutinesBefore := runtime.NumGoroutine()

	for i := 0; i < cycles; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err := f.backend.Pull(ctx, f.target, f.cfg)
		cancel()
		require.NoError(t, err, "pull cycle %d/%d should succeed", i+1, cycles)
	}

	// Give connection teardown and background goroutines a moment to finish.
	deadline := time.Now().Add(5 * time.Second)
	leaked := runtime.NumGoroutine() - goroutinesBefore
	for leaked > goroutineDelta && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
		leaked = runtime.NumGoroutine() - goroutinesBefore
	}
	require.LessOrEqual(t, leaked, goroutineDelta,
		"goroutine count grew by %d after %d pull cycles (before=%d); possible goroutine leak",
		leaked, cycles, goroutinesBefore)
}
