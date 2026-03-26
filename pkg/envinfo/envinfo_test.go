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
	"bytes"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestLogEnvironment(t *testing.T) {
	var buf bytes.Buffer
	logrus.SetOutput(&buf)
	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true})
	defer logrus.SetOutput(nil)

	LogEnvironment(t.TempDir())

	output := buf.String()

	expectedMessages := []string{
		"build info",
		"runtime info",
		"cpu info",
		"memory info",
		"disk info",
	}

	for _, msg := range expectedMessages {
		if !bytes.Contains([]byte(output), []byte(msg)) {
			t.Errorf("expected log output to contain %q, got:\n%s", msg, output)
		}
	}
}

func TestLogDiskInfo(t *testing.T) {
	var buf bytes.Buffer
	logrus.SetOutput(&buf)
	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true})
	defer logrus.SetOutput(nil)

	LogDiskInfo("testDir", t.TempDir())

	output := buf.String()
	if !bytes.Contains([]byte(output), []byte("testDir")) {
		t.Errorf("expected log output to contain 'testDir', got:\n%s", output)
	}
	if !bytes.Contains([]byte(output), []byte("disk info")) {
		t.Errorf("expected log output to contain 'disk info', got:\n%s", output)
	}
}

func TestLogDiskInfoEmptyPath(t *testing.T) {
	var buf bytes.Buffer
	logrus.SetOutput(&buf)
	logrus.SetLevel(logrus.InfoLevel)
	defer logrus.SetOutput(nil)

	LogDiskInfo("empty", "")

	if buf.Len() > 0 {
		t.Errorf("expected no output for empty path, got:\n%s", buf.String())
	}
}

func TestIsVirtualFS(t *testing.T) {
	tests := []struct {
		name     string
		fstype   string
		device   string
		expected bool
	}{
		{"ext4 block device", "ext4", "/dev/sda1", false},
		{"xfs block device", "xfs", "/dev/nvme0n1p1", false},
		{"apfs block device", "apfs", "/dev/disk3s3s1", false},
		{"fuse", "fuse", "s3fs", true},
		{"fuse.s3fs", "fuse.s3fs", "s3fs", true},
		{"fuse.sshfs", "fuse.sshfs", "sshfs#user@host:", true},
		{"nfs", "nfs", "server:/export", true},
		{"nfs4", "nfs", "server:/export", true},
		{"cifs", "cifs", "//server/share", true},
		{"tmpfs", "tmpfs", "tmpfs", true},
		{"overlay", "overlay", "overlay", true},
		{"virtiofs", "virtiofs", "myfs", true},
		{"9p", "9p", "hostshare", true},
		{"unknown non-dev device", "ext4", "some-random-path", true},
		{"empty", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isVirtualFS(tt.fstype, tt.device)
			if result != tt.expected {
				t.Errorf("isVirtualFS(%q, %q) = %v, want %v", tt.fstype, tt.device, result, tt.expected)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    uint64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1048576, "1.00 MB"},
		{1073741824, "1.00 GB"},
		{1099511627776, "1.00 TB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatBytes(tt.input)
			if result != tt.expected {
				t.Errorf("formatBytes(%d) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
