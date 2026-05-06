/*
 *     Copyright 2024 The ModelPack Authors
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
