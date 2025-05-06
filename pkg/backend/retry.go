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
	"time"

	retry "github.com/avast/retry-go/v4"
)

var retryOpts = []retry.Option{
	retry.Attempts(3),
	retry.DelayType(retry.BackOffDelay),
	retry.Delay(1 * time.Second),
	retry.MaxDelay(5 * time.Second),
}
