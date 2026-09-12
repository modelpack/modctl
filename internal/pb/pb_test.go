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

package pb

import (
	"io"
	"sync"
	"testing"
	"time"
)

// TestProgressBarMsgConcurrency exercises concurrent message updates against
// the read path used by mpb's render goroutine. Placeholder and Complete are
// fired from transfer goroutines (e.g. on retry backoff) while the bar is being
// rendered, so progressBar.msg is read and written concurrently. Run with
// -race; before msg was guarded this reproduces the reported data race.
func TestProgressBarMsgConcurrency(t *testing.T) {
	// Render to a discarded writer so the auto-refresh goroutine still runs the
	// prepend decorator (which reads msg) without touching the test output.
	pb := NewProgressBar(io.Discard)
	pb.Start()

	const name = "sha256:deadbeef"
	pb.Add("Copying blob", name, 1024, nil)

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Writers: simulate retry backoff resets (Placeholder) and completion.
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				pb.Placeholder(name, "Copying blob (retry)", 1024)
				pb.Complete(name, "done")
			}
		}()
	}

	// Readers: mirror the render goroutine's read of the message.
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				if b := pb.Get(name); b != nil {
					_ = b.msgText()
				}
			}
		}()
	}

	time.Sleep(150 * time.Millisecond)
	close(stop)
	wg.Wait()

	// Leave the bar completed so Stop()'s mpb.Shutdown() does not block.
	pb.Complete(name, "done")
	pb.Stop()
}
