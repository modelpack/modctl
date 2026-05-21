/*
 *     Copyright 2025 The ModelPack Authors
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
	"bytes"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Functional tests ---

func TestAdd_WrapsReader(t *testing.T) {
	pb := NewProgressBar(io.Discard)
	defer pb.Stop()

	input := "hello world"
	reader := pb.Add("Building =>", "test-file", int64(len(input)), strings.NewReader(input))

	require.NotNil(t, reader)
	var buf bytes.Buffer
	n, err := io.Copy(&buf, reader)
	assert.NoError(t, err)
	assert.Equal(t, int64(len(input)), n)
	assert.Equal(t, input, buf.String())
}

func TestAdd_NilReader(t *testing.T) {
	pb := NewProgressBar(io.Discard)
	defer pb.Stop()

	reader := pb.Add("Checking =>", "test-file", 100, nil)
	assert.Nil(t, reader)

	// Bar should still be created and tracked.
	bar := pb.Get("test-file")
	assert.NotNil(t, bar)
}

func TestAdd_ReplacesExistingBar(t *testing.T) {
	pb := NewProgressBar(io.Discard)
	defer pb.Stop()

	pb.Add("Phase1 =>", "test-file", 100, strings.NewReader("first"))
	bar1 := pb.Get("test-file")
	require.NotNil(t, bar1)

	pb.Add("Phase2 =>", "test-file", 200, strings.NewReader("second"))
	bar2 := pb.Get("test-file")
	require.NotNil(t, bar2)

	// Should be a different bar instance with new size.
	assert.Equal(t, int64(200), bar2.size)
	assert.Equal(t, "Phase2 => test-file", bar2.msg.Load().(string))
}

func TestAdd_DisabledProgress(t *testing.T) {
	SetDisableProgress(true)
	defer SetDisableProgress(false)

	pb := NewProgressBar(io.Discard)
	defer pb.Stop()

	input := strings.NewReader("test")
	reader := pb.Add("Building =>", "test-file", 4, input)
	// When disabled, should return the exact same reader.
	assert.Equal(t, input, reader)
}

func TestReset_SwitchesPhase(t *testing.T) {
	pb := NewProgressBar(io.Discard)
	defer pb.Stop()

	pb.Add("Hashing =>", "test-file", 100, strings.NewReader("hash-data"))

	reader := pb.Reset("Building =>", "test-file", 100, strings.NewReader("build-data"))
	assert.NotNil(t, reader)

	bar := pb.Get("test-file")
	require.NotNil(t, bar)
	assert.Equal(t, "Building => test-file", bar.msg.Load().(string))
}

func TestReset_NoExistingBar(t *testing.T) {
	pb := NewProgressBar(io.Discard)
	defer pb.Stop()

	reader := pb.Reset("Building =>", "new-file", 50, strings.NewReader("data"))
	assert.NotNil(t, reader)

	bar := pb.Get("new-file")
	require.NotNil(t, bar)
	assert.Equal(t, "Building => new-file", bar.msg.Load().(string))
}

// --- Concurrency tests (must pass go test -race) ---

func TestAdd_ConcurrentSameName(t *testing.T) {
	pb := NewProgressBar(io.Discard)
	defer pb.Stop()

	const goroutines = 20
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			pb.Add("Phase =>", "shared-name", 100, strings.NewReader("data"))
		}()
	}

	wg.Wait()

	// Exactly one bar should exist for the name.
	bar := pb.Get("shared-name")
	assert.NotNil(t, bar)
}

func TestComplete_ConcurrentWithRender(t *testing.T) {
	pb := NewProgressBar(io.Discard)
	defer pb.Stop()

	pb.Add("Building =>", "test-file", 100, nil)

	const iterations = 100
	var wg sync.WaitGroup
	wg.Add(2)

	// Goroutine 1: simulate Complete updating msg.
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			pb.Complete("test-file", "Done => test-file")
		}
	}()

	// Goroutine 2: simulate render goroutine reading msg.
	go func() {
		defer wg.Done()
		bar := pb.Get("test-file")
		if bar == nil {
			return
		}
		for i := 0; i < iterations; i++ {
			_ = bar.msg.Load().(string)
		}
	}()

	wg.Wait()
}

func TestAdd_ConcurrentDifferentNames(t *testing.T) {
	pb := NewProgressBar(io.Discard)
	defer pb.Stop()

	const goroutines = 20
	names := make([]string, goroutines)
	for i := 0; i < goroutines; i++ {
		names[i] = strings.Repeat("x", i+1) // unique names
	}

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for _, name := range names {
		go func() {
			defer wg.Done()
			pb.Add("Building =>", name, 100, strings.NewReader("data"))
		}()
	}

	wg.Wait()

	// All bars should exist.
	for _, name := range names {
		bar := pb.Get(name)
		assert.NotNil(t, bar, "bar for %q should exist", name)
	}
}

// --- Error / idempotency tests ---

func TestAbort_NonExistentBar(t *testing.T) {
	pb := NewProgressBar(io.Discard)
	defer pb.Stop()

	// Should not panic.
	pb.Abort("does-not-exist", errors.New("test error"))
}

func TestAbort_AlreadyAbortedBar(t *testing.T) {
	pb := NewProgressBar(io.Discard)
	defer pb.Stop()

	pb.Add("Building =>", "test-file", 100, nil)
	pb.Abort("test-file", errors.New("first abort"))

	// Second abort should not panic (mpb uses sync.Once internally).
	pb.Abort("test-file", errors.New("second abort"))
}

func TestComplete_NonExistentBar(t *testing.T) {
	pb := NewProgressBar(io.Discard)
	defer pb.Stop()

	// Should not panic.
	pb.Complete("does-not-exist", "done")
}

func TestAdd_AfterAbort(t *testing.T) {
	pb := NewProgressBar(io.Discard)
	defer pb.Stop()

	pb.Add("Phase1 =>", "test-file", 100, nil)
	pb.Abort("test-file", errors.New("abort"))

	// Add same name again should work.
	reader := pb.Add("Phase2 =>", "test-file", 200, strings.NewReader("data"))
	assert.NotNil(t, reader)

	bar := pb.Get("test-file")
	require.NotNil(t, bar)
	assert.Equal(t, int64(200), bar.size)
}
