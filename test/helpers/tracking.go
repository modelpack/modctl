package helpers

import (
	"io"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TrackingReadCloser wraps an io.ReadCloser and records whether Close() was called.
type TrackingReadCloser struct {
	io.ReadCloser
	closed atomic.Bool
}

// NewTrackingReadCloser wraps rc with close-tracking.
func NewTrackingReadCloser(rc io.ReadCloser) *TrackingReadCloser {
	return &TrackingReadCloser{ReadCloser: rc}
}

// Close marks the closer as closed and delegates to the underlying closer.
func (t *TrackingReadCloser) Close() error {
	t.closed.Store(true)
	return t.ReadCloser.Close()
}

// WasClosed returns true if Close() was called.
func (t *TrackingReadCloser) WasClosed() bool {
	return t.closed.Load()
}

// AssertClosed asserts Close() was called.
func (t *TrackingReadCloser) AssertClosed(tb testing.TB) {
	tb.Helper()
	assert.True(tb, t.closed.Load(), "ReadCloser was not closed")
}

// AssertNotClosed asserts Close() was NOT called (for reverse assertions on known bugs).
func (t *TrackingReadCloser) AssertNotClosed(tb testing.TB) {
	tb.Helper()
	assert.False(tb, t.closed.Load(), "ReadCloser was unexpectedly closed — bug may be fixed!")
}
