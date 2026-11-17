//go:build slowtest

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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/modelpack/modctl/test/helpers"
)

// TestSlow_Pull_RetryOnTransientError verifies that a pull succeeds when the
// first 2 requests for a layer blob fail transiently (FailOnNthRequest: 2)
// and the retry mechanism eventually succeeds.  Requires real backoff —
// takes 15+ seconds.
func TestSlow_Pull_RetryOnTransientError(t *testing.T) {
	f := newPullTestFixture(t, 1)
	defer f.mr.Close()

	// Only the layer blob fails: the first 2 GETs return 500, the 3rd
	// succeeds. The fault is scoped to the blob path because the manifest
	// fetch runs before the retry loop, so a global fault would fail the
	// pull immediately instead of exercising the retry.
	f.mr.WithFault(&helpers.FaultConfig{
		PathFaults: map[string]*helpers.FaultConfig{
			f.digests[0]: {FailOnNthRequest: 2},
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	err := f.backend.Pull(ctx, f.target, f.cfg)
	require.NoError(t, err, "pull should eventually succeed after transient failures")
}

// TestSlow_Pull_RetryExhausted verifies that a pull fails when every request
// for a layer blob returns 500: the retry loop keeps backing off until the
// context deadline and the error is reported.  Requires real backoff — takes
// 60 seconds.
func TestSlow_Pull_RetryExhausted(t *testing.T) {
	f := newPullTestFixture(t, 1)
	defer f.mr.Close()

	// Every GET for the layer blob returns 500. The fault is path-scoped so
	// the manifest fetch succeeds and the blob retry loop is exercised.
	f.mr.WithFault(&helpers.FaultConfig{
		PathFaults: map[string]*helpers.FaultConfig{
			f.digests[0]: {StatusCodeOverride: 500},
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	err := f.backend.Pull(ctx, f.target, f.cfg)
	require.Error(t, err, "pull should fail when all retries are exhausted")
}

// TestSlow_Pull_RateLimited verifies that a pull succeeds when the first 3
// requests for a layer blob fail (simulating rate-limiting) and subsequent
// requests succeed.  Requires real backoff — takes 35+ seconds.
func TestSlow_Pull_RateLimited(t *testing.T) {
	f := newPullTestFixture(t, 1)
	defer f.mr.Close()

	// The first 3 GETs for the layer blob fail; the 4th succeeds. See
	// TestSlow_Pull_RetryOnTransientError for why the fault is path-scoped.
	f.mr.WithFault(&helpers.FaultConfig{
		PathFaults: map[string]*helpers.FaultConfig{
			f.digests[0]: {FailOnNthRequest: 3},
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	err := f.backend.Pull(ctx, f.target, f.cfg)
	require.NoError(t, err, "pull should eventually succeed after rate-limit simulation")
}

// TestIntegration_Pull_AuthErrorFailsFast verifies that 401 auth errors are
// not retried: Pull must fail immediately instead of burning the full retry
// backoff. Regression test for https://github.com/modelpack/modctl/issues/494.
func TestIntegration_Pull_AuthErrorFailsFast(t *testing.T) {
	f := newPullTestFixture(t, 1)
	defer f.mr.Close()

	// Every request returns 401; the retry loop must give up immediately.
	f.mr.WithFault(&helpers.FaultConfig{
		StatusCodeOverride: 401,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	start := time.Now()
	err := f.backend.Pull(ctx, f.target, f.cfg)
	elapsed := time.Since(start)

	require.Error(t, err, "pull must fail on 401")
	// #494: auth errors fail fast. With retries this would take 30+ seconds.
	assert.Less(t, elapsed, 5*time.Second,
		"auth errors must not be retried (#494)")
}
