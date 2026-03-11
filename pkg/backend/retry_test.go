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
	"errors"
	"testing"

	retry "github.com/avast/retry-go/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testRetryOpts overrides defaultRetryOpts with no delay for fast tests.
func testRetryOpts(ctx context.Context, maxAttempts uint) []retry.Option {
	return []retry.Option{
		retry.Attempts(maxAttempts),
		retry.Delay(0),
		retry.MaxDelay(0),
		retry.Context(ctx),
	}
}

func TestRetrySuccessAfterFailures(t *testing.T) {
	ctx := context.Background()
	attempts := 0

	err := retry.Do(func() error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary failure")
		}
		return nil
	}, testRetryOpts(ctx, 6)...)

	require.NoError(t, err)
	assert.Equal(t, 3, attempts)
}

func TestRetryMaxAttemptsExceeded(t *testing.T) {
	ctx := context.Background()
	attempts := 0
	maxAttempts := uint(4)

	err := retry.Do(func() error {
		attempts++
		return errors.New("persistent failure")
	}, testRetryOpts(ctx, maxAttempts)...)

	assert.Error(t, err)
	assert.Equal(t, int(maxAttempts), attempts)
}

func TestRetryStopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0

	err := retry.Do(func() error {
		attempts++
		if attempts == 2 {
			cancel()
		}
		return errors.New("temporary failure")
	}, testRetryOpts(ctx, 10)...)

	assert.Error(t, err)
	assert.True(t, attempts >= 2, "should have attempted at least 2 times")
}

func TestRetrySucceedsOnFirstAttempt(t *testing.T) {
	ctx := context.Background()
	attempts := 0

	err := retry.Do(func() error {
		attempts++
		return nil
	}, testRetryOpts(ctx, 6)...)

	require.NoError(t, err)
	assert.Equal(t, 1, attempts)
}
