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
	"fmt"
	"time"

	internalpb "github.com/modelpack/modctl/internal/pb"
)

// newRetryPlaceholder returns a retrypolicy OnRetry callback that keeps a
// progress bar visible during backoff, rendering a consistent
// "<prompt> (retry N, <reason>, waiting <backoff>) <name>" line across every
// transfer path (push, pull, fetch, and their Dragonfly variants). Previously
// each path formatted retries differently — some showed the attempt/reason,
// others silently reset the bar — which made the retry UX inconsistent.
//
// name is the bar key (the layer digest); prompt is the already-normalized base
// prompt (e.g. internalpb.NormalizePrompt("Pulling blob")).
func newRetryPlaceholder(pb *internalpb.ProgressBar, name, prompt string, size int64) func(uint, string, time.Duration) {
	return func(attempt uint, reason string, backoff time.Duration) {
		detail := fmt.Sprintf("%s (retry %d, %s, waiting %s)", prompt, attempt, reason, backoff.Truncate(time.Second))
		pb.Placeholder(name, detail, size)
	}
}
