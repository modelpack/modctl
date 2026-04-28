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

//go:build linux

package envinfo

import (
	"testing"
)

func TestGetCgroupLimits(t *testing.T) {
	// This test runs on Linux and verifies getCgroupLimits does not panic.
	// The actual values depend on the environment (container vs bare metal).
	limits := getCgroupLimits()
	if limits == nil {
		t.Fatal("getCgroupLimits returned nil")
	}

	t.Logf("InCgroup=%v CPUQuota=%.2f MemLimit=%d", limits.InCgroup, limits.CPUQuota, limits.MemLimit)
}
