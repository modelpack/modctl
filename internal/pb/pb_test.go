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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// lockedBuffer is a bytes.Buffer that is safe for concurrent use. The mpb
// render goroutine writes frames while the test reads them.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// waitForOutput polls the buffer until it contains want or the timeout
// expires, and returns the final contents.
func waitForOutput(t *testing.T, out *lockedBuffer, want string, timeout time.Duration) string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if s := out.String(); strings.Contains(s, want) {
			return s
		}
		time.Sleep(50 * time.Millisecond)
	}
	s := out.String()
	require.Contains(t, s, want, "progress output never rendered %q", want)
	return s
}

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

func TestSpinner_CreatesIndeterminateBar(t *testing.T) {
	pb := NewProgressBar(io.Discard)
	defer pb.Stop()

	pb.Spinner("Checking =>", "test-file", 100)

	bar := pb.Get("test-file")
	require.NotNil(t, bar)
	assert.True(t, bar.indeterminate)
	assert.Equal(t, int64(100), bar.size)
	assert.Equal(t, "Checking => test-file", bar.msg.Load().(string))
	assert.False(t, bar.Completed(), "spinner must not complete on its own")
}

func TestSpinner_ResetSwitchesToTransferBar(t *testing.T) {
	pb := NewProgressBar(io.Discard)
	defer pb.Stop()

	pb.Spinner("Checking =>", "test-file", 100)
	spinner := pb.Get("test-file")
	require.NotNil(t, spinner)

	reader := pb.Reset("Pushing =>", "test-file", 100, strings.NewReader("data"))
	require.NotNil(t, reader)

	bar := pb.Get("test-file")
	require.NotNil(t, bar)
	assert.NotSame(t, spinner, bar)
	assert.False(t, bar.indeterminate)
	assert.Equal(t, "Pushing => test-file", bar.msg.Load().(string))
	assert.True(t, spinner.Aborted(), "replaced spinner must be aborted")
}

func TestSpinner_CompleteFinishesBar(t *testing.T) {
	pb := NewProgressBar(io.Discard)
	defer pb.Stop()

	pb.Spinner("Checking =>", "test-file", 100)
	pb.Complete("test-file", "Skipped => test-file")

	bar := pb.Get("test-file")
	require.NotNil(t, bar)
	assert.Equal(t, "Skipped => test-file", bar.msg.Load().(string))
	assert.Eventually(t, bar.Completed, 2*time.Second, 20*time.Millisecond,
		"Complete must finish an indeterminate bar")
}

func TestSpinner_DisabledProgress(t *testing.T) {
	SetDisableProgress(true)
	defer SetDisableProgress(false)

	pb := NewProgressBar(io.Discard)
	defer pb.Stop()

	pb.Spinner("Checking =>", "test-file", 100)
	assert.Nil(t, pb.Get("test-file"), "no bar must be tracked when progress is disabled")
}

func TestSpinner_RendersWithoutTransferRate(t *testing.T) {
	out := &lockedBuffer{}
	pb := NewProgressBar(out)

	pb.Spinner("Checking =>", "test-file", 100)
	rendered := waitForOutput(t, out, "Checking => test-file", 3*time.Second)
	pb.Stop()

	// The spinner shows the size and elapsed time, but no byte counter and
	// no transfer rate. A transfer bar with no bytes flowing renders
	// "0.00 b / 100.00 b | 0.00 b/s" (see TestAdd_RendersTransferRate).
	assert.Contains(t, rendered, "100 B")
	assert.NotContains(t, rendered, "b/s")
	assert.NotContains(t, rendered, "0.00 b / 100.00 b")
}

func TestAdd_RendersTransferRate(t *testing.T) {
	out := &lockedBuffer{}
	pb := NewProgressBar(out)

	pb.Add("Pushing =>", "test-file", 100, nil)
	rendered := waitForOutput(t, out, "Pushing => test-file", 3*time.Second)
	pb.Stop()

	// Sanity check for the assertion above: a transfer bar does render the
	// counter and the rate, so the spinner test is not vacuous.
	assert.Contains(t, rendered, "0.00 b / 100.00 b")
	assert.Contains(t, rendered, "0.00 b/s")
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

func TestReset_AfterCompletedPhase(t *testing.T) {
	pb := NewProgressBar(io.Discard)
	defer pb.Stop()

	// Phase 1: fully consume the reader so the bar reaches complete state
	// (mpb ignores Abort on completed bars; PopCompletedMode pops them).
	data := strings.Repeat("a", 100)
	reader := pb.Add("Hashing =>", "test-file", 100, strings.NewReader(data))
	n, err := io.Copy(io.Discard, reader)
	require.NoError(t, err)
	require.Equal(t, int64(100), n)

	// Phase 2: Reset must still create a fresh bar and replace the entry.
	reader2 := pb.Reset("Building =>", "test-file", 100, strings.NewReader(data))
	require.NotNil(t, reader2)

	bar := pb.Get("test-file")
	require.NotNil(t, bar)
	assert.Equal(t, "Building => test-file", bar.msg.Load().(string))

	// New bar must be fully usable.
	n, err = io.Copy(io.Discard, reader2)
	assert.NoError(t, err)
	assert.Equal(t, int64(100), n)
}
