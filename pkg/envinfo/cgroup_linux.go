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
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/sirupsen/logrus"
)

// cgroupLimits holds the CPU and memory limits from cgroup.
type cgroupLimits struct {
	CPUQuota float64 // effective CPU cores (quota/period), 0 if unlimited
	MemLimit uint64  // memory limit in bytes, 0 if unlimited
	InCgroup bool    // true if running inside a cgroup with limits
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

// resolveCgroupV2Path returns the filesystem path to the current process's
// cgroup v2 directory by reading /proc/self/cgroup. In cgroup v2 (unified
// hierarchy), the file contains a single line like "0::/kubepods/pod-xxx".
// Returns empty string if the cgroup path cannot be determined.
func resolveCgroupV2Path() string {
	data, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		// cgroup v2 line format: "0::<path>"
		parts := strings.SplitN(line, ":", 3)
		if len(parts) == 3 && parts[0] == "0" && parts[1] == "" {
			cgPath := parts[2]
			if cgPath == "" || cgPath == "/" {
				// Root cgroup — fall back to /sys/fs/cgroup directly.
				return "/sys/fs/cgroup"
			}
			return filepath.Join("/sys/fs/cgroup", cgPath)
		}
	}

	return ""
}

func tryV2(limits *cgroupLimits) bool {
	cgDir := resolveCgroupV2Path()
	if cgDir == "" {
		return false
	}

	// Detect CPU and memory independently — one may be set without the other.
	if cpuMax, err := os.ReadFile(filepath.Join(cgDir, "cpu.max")); err == nil {
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

	if memMax, err := os.ReadFile(filepath.Join(cgDir, "memory.max")); err == nil {
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

// resolveCgroupV1Path returns the filesystem path for a given cgroup v1
// controller (e.g., "cpu", "memory") by reading /proc/self/cgroup.
// Each line has format: "hierarchy-ID:controller-list:cgroup-path".
func resolveCgroupV1Path(controller string) string {
	data, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		parts := strings.SplitN(line, ":", 3)
		if len(parts) != 3 {
			continue
		}

		controllers := strings.Split(parts[1], ",")
		for _, c := range controllers {
			if c == controller {
				cgPath := parts[2]
				return filepath.Join("/sys/fs/cgroup", controller, cgPath)
			}
		}
	}

	// Fallback to the base controller path.
	return filepath.Join("/sys/fs/cgroup", controller)
}

func tryV1(limits *cgroupLimits) {
	// cgroup v1: CPU quota. Detect independently from memory.
	cpuDir := resolveCgroupV1Path("cpu")
	if quotaBytes, err := os.ReadFile(filepath.Join(cpuDir, "cpu.cfs_quota_us")); err == nil {
		if quota, err := strconv.ParseFloat(strings.TrimSpace(string(quotaBytes)), 64); err == nil && quota > 0 {
			if periodBytes, err := os.ReadFile(filepath.Join(cpuDir, "cpu.cfs_period_us")); err == nil {
				if period, err := strconv.ParseFloat(strings.TrimSpace(string(periodBytes)), 64); err == nil && period > 0 {
					limits.CPUQuota = quota / period
					limits.InCgroup = true
				}
			}
		}
	}

	// cgroup v1: Memory limit.
	memDir := resolveCgroupV1Path("memory")
	if memBytes, err := os.ReadFile(filepath.Join(memDir, "memory.limit_in_bytes")); err == nil {
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
