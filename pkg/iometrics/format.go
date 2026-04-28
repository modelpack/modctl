/*
 *     Copyright 2024 The CNAI Authors
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

package iometrics

import (
	"fmt"
	"time"
)

func formatThroughput(bytes int64, d time.Duration) string {
	if d == 0 || bytes == 0 {
		return "N/A"
	}
	mbPerSec := float64(bytes) / (1024 * 1024) / d.Seconds()
	return fmt.Sprintf("%.2f MB/s", mbPerSec)
}
