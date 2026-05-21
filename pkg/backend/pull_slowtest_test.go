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

//go:build slowtest

package backend

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/modelpack/modctl/test/helpers"
)

// TestSlow_Pull_RetryOnTransientError verifies that a pull succeeds when the
// first 2 requests fail transiently (FailOnNthRequest: 2) and the retry
// mechanism eventually succeeds.  Requires real backoff — takes 30+ seconds.
func TestSlow_Pull_RetryOnTransientError(t *testing.T) {
	f := newPullTestFixture(t, 1)
	defer f.mr.Close()

	// First 2 requests fail with 500; request 3+ succeed normally.
	// The failCounter is global across all requests (ping, manifest, blobs),
	// so a value of 2 means the ping and one other request fail before success.
	f.mr.WithFault(&helpers.FaultConfig{
		FailOnNthRequest: 2,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	err := f.backend.Pull(ctx, f.target, f.cfg)
	require.NoError(t, err, "pull should eventually succeed after transient failures")
}

// TestSlow_Pull_RetryExhausted verifies that a pull fails when all registry
// requests return 500, causing all retry attempts to be exhausted.  Requires
// real backoff — takes 30+ seconds.
func TestSlow_Pull_RetryExhausted(t *testing.T) {
	f := newPullTestFixture(t, 1)
	defer f.mr.Close()

	// Every request returns 500, so all retry attempts will be exhausted.
	f.mr.WithFault(&helpers.FaultConfig{
		StatusCodeOverride: 500,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	err := f.backend.Pull(ctx, f.target, f.cfg)
	require.Error(t, err, "pull should fail when all retries are exhausted")
}

// TestSlow_Pull_RateLimited verifies that a pull succeeds when the first 3
// requests fail (simulating rate-limiting) and subsequent requests succeed.
// Requires real backoff — takes 30+ seconds.
func TestSlow_Pull_RateLimited(t *testing.T) {
	f := newPullTestFixture(t, 1)
	defer f.mr.Close()

	// First 3 requests fail; request 4+ succeed normally.
	f.mr.WithFault(&helpers.FaultConfig{
		FailOnNthRequest: 3,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	err := f.backend.Pull(ctx, f.target, f.cfg)
	require.NoError(t, err, "pull should eventually succeed after rate-limit simulation")
}

// TestKnownBug_Pull_AuthErrorStillRetries documents that 401 auth errors are
// currently retried with full backoff instead of failing immediately.
// See: https://github.com/modelpack/modctl/issues/494
//
// REVERSE ASSERTION: today this test passes because elapsed > 10s (retries
// occur with backoff before exhaustion).  When #494 is fixed, auth errors
// should fail immediately (<2s) — flip the assertion to:
//
//	assert.Less(t, elapsed, 5*time.Second)
func TestKnownBug_Pull_AuthErrorStillRetries(t *testing.T) {
	f := newPullTestFixture(t, 1)
	defer f.mr.Close()

	// Every request returns 401; the retry loop should not give up immediately.
	f.mr.WithFault(&helpers.FaultConfig{
		StatusCodeOverride: 401,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	start := time.Now()
	_ = f.backend.Pull(ctx, f.target, f.cfg)
	elapsed := time.Since(start)

	// Known bug #494: auth errors ARE retried, wasting 30+ seconds of backoff.
	// Reverse assertion — passes today because retries happen.
	// When #494 is fixed, auth errors should fail immediately (<2s).
	// Flip assertion to: assert.Less(t, elapsed, 5*time.Second)
	assert.Greater(t, elapsed, 10*time.Second,
		"auth errors should be retried (known bug #494); elapsed time documents backoff duration")
}
