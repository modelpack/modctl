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

package backend

import (
	"errors"
	"net/http"
	"time"

	retry "github.com/avast/retry-go/v4"
	"oras.land/oras-go/v2/registry/remote/errcode"
)

// isAuthError reports whether err represents a registry auth failure that
// cannot be fixed by retrying the same request with the same credentials.
// Retrying those wastes the user's time on the ~30s+ exponential backoff
// before the final error surfaces, so callers should short-circuit instead.
func isAuthError(err error) bool {
	var respErr *errcode.ErrorResponse
	if errors.As(err, &respErr) {
		return respErr.StatusCode == http.StatusUnauthorized ||
			respErr.StatusCode == http.StatusForbidden
	}
	return false
}

var defaultRetryOpts = []retry.Option{
	retry.Attempts(6),
	retry.DelayType(retry.BackOffDelay),
	retry.Delay(5 * time.Second),
	retry.MaxDelay(60 * time.Second),
	// Registry auth errors will not recover on retry; fail fast so the user
	// sees the real error within seconds instead of 30s+ of silent backoff.
	retry.RetryIf(func(err error) bool { return !isAuthError(err) }),
}
