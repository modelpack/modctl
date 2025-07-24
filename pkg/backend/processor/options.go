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

package processor

import (
	"time"

	retry "github.com/avast/retry-go/v4"

	"github.com/modelpack/modctl/internal/pb"
)

type ProcessOption func(*processOptions)

type processOptions struct {
	// concurrency is the number of concurrent workers to use for processing.
	concurrency int
	// progressTracker is the progress bar to use for tracking progress.
	progressTracker *pb.ProgressBar
}

func WithConcurrency(concurrency int) ProcessOption {
	return func(o *processOptions) {
		o.concurrency = concurrency
	}
}

func WithProgressTracker(tracker *pb.ProgressBar) ProcessOption {
	return func(o *processOptions) {
		o.progressTracker = tracker
	}
}

var defaultRetryOpts = []retry.Option{
	retry.Attempts(4),
	retry.DelayType(retry.BackOffDelay),
	retry.Delay(10 * time.Second),
	retry.MaxDelay(20 * time.Second),
}
