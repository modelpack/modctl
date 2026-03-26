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

package envinfo

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/sirupsen/logrus"

	"github.com/modelpack/modctl/pkg/version"
)

// LogEnvironment collects and logs system environment information.
// Individual collection failures are logged as warnings and do not
// prevent other information from being collected.
func LogEnvironment(storageDir string) {
	logVersionInfo()
	logRuntimeInfo()
	logCPUInfo()
	logMemoryInfo()
	logCgroupInfo()
	LogDiskInfo("storageDir", storageDir)
}

// LogDiskInfo logs disk usage information for the device that hosts
// the given path. It can be called from any command to log disk info
// for command-specific directories.
func LogDiskInfo(name, path string) {
	if path == "" {
		return
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		logrus.WithError(err).Warnf("failed to resolve path for %s", name)
		return
	}

	usage, err := disk.Usage(absPath)
	if err != nil {
		logrus.WithError(err).Warnf("failed to get disk usage for %s", name)
		return
	}

	logrus.WithFields(logrus.Fields{
		"name":         name,
		"path":         absPath,
		"fstype":       usage.Fstype,
		"total":        formatBytes(usage.Total),
		"free":         formatBytes(usage.Free),
		"usagePercent": fmt.Sprintf("%.1f%%", usage.UsedPercent),
	}).Info("disk info")
}

func logVersionInfo() {
	logrus.WithFields(logrus.Fields{
		"version":   version.GitVersion,
		"commit":    version.GitCommit,
		"platform":  version.Platform,
		"buildTime": version.BuildTime,
	}).Info("build info")
}

func logRuntimeInfo() {
	logrus.WithFields(logrus.Fields{
		"go":         runtime.Version(),
		"os":         runtime.GOOS,
		"arch":       runtime.GOARCH,
		"gomaxprocs": runtime.GOMAXPROCS(0),
	}).Info("runtime info")
}

func logCPUInfo() {
	physicalCount, err := cpu.Counts(false)
	if err != nil {
		logrus.WithError(err).Warn("failed to get physical CPU count")
	}

	logicalCount, err := cpu.Counts(true)
	if err != nil {
		logrus.WithError(err).Warn("failed to get logical CPU count")
	}

	fields := logrus.Fields{
		"physicalCores": physicalCount,
		"logicalCores":  logicalCount,
	}

	infos, err := cpu.Info()
	if err != nil {
		logrus.WithError(err).Warn("failed to get CPU model info")
	} else if len(infos) > 0 {
		fields["model"] = infos[0].ModelName
	}

	percents, err := cpu.Percent(0, false)
	if err != nil {
		logrus.WithError(err).Warn("failed to get CPU usage")
	} else if len(percents) > 0 {
		fields["usagePercent"] = fmt.Sprintf("%.1f%%", percents[0])
	}

	logrus.WithFields(fields).Info("cpu info")
}

func logMemoryInfo() {
	v, err := mem.VirtualMemory()
	if err != nil {
		logrus.WithError(err).Warn("failed to get memory info")
		return
	}

	logrus.WithFields(logrus.Fields{
		"total":        formatBytes(v.Total),
		"available":    formatBytes(v.Available),
		"usagePercent": fmt.Sprintf("%.1f%%", v.UsedPercent),
	}).Info("memory info")
}

// isVirtualFS checks whether a filesystem type or device path indicates
// a non-block-device filesystem where IO counters are not available.
// This includes FUSE, network filesystems, RAM-based filesystems, and
// container overlay filesystems.
func isVirtualFS(fstype, device string) bool {
	fstypeLower := strings.ToLower(fstype)

	virtualFSTypes := []string{
		"fuse", "nfs", "cifs", "smb", "smbfs",
		"tmpfs", "ramfs", "devtmpfs",
		"overlay", "aufs",
		"sshfs", "s3fs", "gcsfuse", "ossfs",
		"9p", "virtiofs",
	}

	for _, vfs := range virtualFSTypes {
		if fstypeLower == vfs || strings.HasPrefix(fstypeLower, vfs+".") {
			return true
		}
	}

	// Devices not under /dev/ are generally not block devices.
	// e.g., "s3fs", "sshfs#user@host:", "server:/export"
	if device != "" && !strings.HasPrefix(device, "/dev/") {
		return true
	}

	return false
}

func formatBytes(b uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case b >= TB:
		return fmt.Sprintf("%.2f TB", float64(b)/float64(TB))
	case b >= GB:
		return fmt.Sprintf("%.2f GB", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.2f MB", float64(b)/float64(MB))
	case b >= KB:
		return fmt.Sprintf("%.2f KB", float64(b)/float64(KB))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
