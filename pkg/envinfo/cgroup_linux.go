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
	cpuMax, err := os.ReadFile("/sys/fs/cgroup/cpu.max")
	if err != nil {
		return false
	}

	parts := strings.Fields(strings.TrimSpace(string(cpuMax)))
	if len(parts) == 2 && parts[0] != "max" {
		quota, err1 := strconv.ParseFloat(parts[0], 64)
		period, err2 := strconv.ParseFloat(parts[1], 64)
		if err1 == nil && err2 == nil && period > 0 {
			limits.CPUQuota = quota / period
			limits.InCgroup = true
		}
	}

	memMax, err := os.ReadFile("/sys/fs/cgroup/memory.max")
	if err != nil {
		return limits.InCgroup
	}

	memStr := strings.TrimSpace(string(memMax))
	if memStr != "max" {
		memLimit, err := strconv.ParseUint(memStr, 10, 64)
		if err == nil {
			limits.MemLimit = memLimit
			limits.InCgroup = true
		}
	}

	return limits.InCgroup
}

func tryV1(limits *cgroupLimits) {
	// cgroup v1: CPU quota.
	quotaBytes, err := os.ReadFile("/sys/fs/cgroup/cpu/cpu.cfs_quota_us")
	if err != nil {
		return
	}

	quota, err := strconv.ParseFloat(strings.TrimSpace(string(quotaBytes)), 64)
	if err != nil || quota <= 0 {
		// -1 means no limit.
		return
	}

	periodBytes, err := os.ReadFile("/sys/fs/cgroup/cpu/cpu.cfs_period_us")
	if err != nil {
		return
	}

	period, err := strconv.ParseFloat(strings.TrimSpace(string(periodBytes)), 64)
	if err != nil || period <= 0 {
		return
	}

	limits.CPUQuota = quota / period
	limits.InCgroup = true

	// cgroup v1: Memory limit.
	memBytes, err := os.ReadFile("/sys/fs/cgroup/memory/memory.limit_in_bytes")
	if err != nil {
		return
	}

	memLimit, err := strconv.ParseUint(strings.TrimSpace(string(memBytes)), 10, 64)
	if err != nil {
		return
	}

	// Very large values (like 2^63) indicate no limit.
	const noLimitThreshold = 1 << 62
	if memLimit < noLimitThreshold {
		limits.MemLimit = memLimit
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
		fields["memoryLimit"] = formatBytes(limits.MemLimit)
	}

	if len(fields) > 0 {
		logrus.WithFields(fields).Info("cgroup limits")
	}
}
