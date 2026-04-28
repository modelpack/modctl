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

	"github.com/stretchr/testify/require"

	"github.com/modelpack/modctl/test/helpers"
)

// TestSlow_Push_RetryOnTransientError verifies that a push succeeds when the
// blob upload initiation endpoint (/blobs/uploads/) fails on the first 2
// attempts and then succeeds.  Requires real backoff — takes 30+ seconds.
func TestSlow_Push_RetryOnTransientError(t *testing.T) {
	f := newPushTestFixture(t)
	defer f.mr.Close()

	// Fail the first 2 requests to the blob upload POST endpoint; request 3+
	// succeed normally, allowing the push to complete via retry.
	f.mr.WithFault(&helpers.FaultConfig{
		PathFaults: map[string]*helpers.FaultConfig{
			"/blobs/uploads/": {FailOnNthRequest: 2},
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	err := f.backend.Push(ctx, f.target, f.cfg)
	require.NoError(t, err, "push should eventually succeed after transient blob upload failures")
}
