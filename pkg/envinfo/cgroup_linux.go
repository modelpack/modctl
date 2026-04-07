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
	"os"
	"strconv"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/sirupsen/logrus"
)

// cgroupLimits holds the CPU and memory limits from cgroup.
type cgroupLimits struct {
	CPUQuota  float64 // effective CPU cores (quota/period), 0 if unlimited
	MemLimit  uint64  // memory limit in bytes, 0 if unlimited
	InCgroup  bool    // true if running inside a cgroup with limits
}

// getCgroupLimits reads cgroup v2 then v1 CPU and memory limits.
func getCgroupLimits() *cgroupLimits {
	limits := &cgroupLimits{}

	// Try cgroup v2 first.
	if tryV2(limits) {
		return limits
	}

	// Fall back to cgroup v1.
	tryV1(limits)
	return limits
}

func tryV2(limits *cgroupLimits) bool {
	// cgroup v2 uses unified hierarchy at /sys/fs/cgroup/.
	// Detect CPU and memory independently — one may be set without the other.
	if cpuMax, err := os.ReadFile("/sys/fs/cgroup/cpu.max"); err == nil {
		parts := strings.Fields(strings.TrimSpace(string(cpuMax)))
		if len(parts) == 2 && parts[0] != "max" {
			quota, err1 := strconv.ParseFloat(parts[0], 64)
			period, err2 := strconv.ParseFloat(parts[1], 64)
			if err1 == nil && err2 == nil && period > 0 {
				limits.CPUQuota = quota / period
				limits.InCgroup = true
			}
		}
	}

	if memMax, err := os.ReadFile("/sys/fs/cgroup/memory.max"); err == nil {
		memStr := strings.TrimSpace(string(memMax))
		if memStr != "max" {
			if memLimit, err := strconv.ParseUint(memStr, 10, 64); err == nil {
				limits.MemLimit = memLimit
				limits.InCgroup = true
			}
		}
	}

	return limits.InCgroup
}

func tryV1(limits *cgroupLimits) {
	// cgroup v1: CPU quota. Detect independently from memory.
	if quotaBytes, err := os.ReadFile("/sys/fs/cgroup/cpu/cpu.cfs_quota_us"); err == nil {
		if quota, err := strconv.ParseFloat(strings.TrimSpace(string(quotaBytes)), 64); err == nil && quota > 0 {
			if periodBytes, err := os.ReadFile("/sys/fs/cgroup/cpu/cpu.cfs_period_us"); err == nil {
				if period, err := strconv.ParseFloat(strings.TrimSpace(string(periodBytes)), 64); err == nil && period > 0 {
					limits.CPUQuota = quota / period
					limits.InCgroup = true
				}
			}
		}
	}

	// cgroup v1: Memory limit.
	if memBytes, err := os.ReadFile("/sys/fs/cgroup/memory/memory.limit_in_bytes"); err == nil {
		if memLimit, err := strconv.ParseUint(strings.TrimSpace(string(memBytes)), 10, 64); err == nil {
			// Very large values (like 2^63) indicate no limit.
			const noLimitThreshold = 1 << 62
			if memLimit < noLimitThreshold {
				limits.MemLimit = memLimit
				limits.InCgroup = true
			}
		}
	}
}

// logCgroupInfo logs container resource limits if running in a cgroup.
func logCgroupInfo() {
	limits := getCgroupLimits()
	if !limits.InCgroup {
		return
	}

	fields := logrus.Fields{}
	if limits.CPUQuota > 0 {
		fields["cpuQuota"] = strconv.FormatFloat(limits.CPUQuota, 'f', 2, 64)
	}
	if limits.MemLimit > 0 {
		fields["memoryLimit"] = humanize.IBytes(limits.MemLimit)
	}

	if len(fields) > 0 {
		logrus.WithFields(fields).Info("cgroup limits")
	}
}
