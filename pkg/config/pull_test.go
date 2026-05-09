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

package config

import (
	"errors"
	"testing"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
)

func TestNewPull_DefaultsAndValidate(t *testing.T) {
	p := NewPull()
	assert.NotNil(t, p.Hooks, "default hooks must be non-nil")
	assert.NoError(t, p.Validate())

	// emptyPullHook never asks the backend to skip and accepts results
	// without panicking.
	desc := ocispec.Descriptor{Digest: "sha256:deadbeef"}
	assert.False(t, p.Hooks.BeforePullLayer(desc, ocispec.Manifest{}),
		"default hook must not skip layers")
	p.Hooks.AfterPullLayer(desc, false, nil)
	p.Hooks.AfterPullLayer(desc, true, nil)
	p.Hooks.AfterPullLayer(desc, false, errors.New("boom"))
}

// recordingHook is a PullHooks implementation used to verify the interface
// contract: BeforePullLayer can request a skip, and AfterPullLayer reports
// whether the layer was skipped along with any error.
type recordingHook struct {
	skipDigests map[string]bool

	beforeCalls []ocispec.Descriptor
	afterCalls  []afterCall
}

type afterCall struct {
	desc    ocispec.Descriptor
	skipped bool
	err     error
}

func (r *recordingHook) BeforePullLayer(desc ocispec.Descriptor, _ ocispec.Manifest) bool {
	r.beforeCalls = append(r.beforeCalls, desc)
	return r.skipDigests[desc.Digest.String()]
}

func (r *recordingHook) AfterPullLayer(desc ocispec.Descriptor, skipped bool, err error) {
	r.afterCalls = append(r.afterCalls, afterCall{desc: desc, skipped: skipped, err: err})
}

func TestPullHooks_InterfaceContract(t *testing.T) {
	// Compile-time check that recordingHook satisfies the PullHooks interface.
	var _ PullHooks = (*recordingHook)(nil)

	hook := &recordingHook{
		skipDigests: map[string]bool{"sha256:aaa": true},
	}
	a := ocispec.Descriptor{Digest: "sha256:aaa"}
	b := ocispec.Descriptor{Digest: "sha256:bbb"}

	assert.True(t, hook.BeforePullLayer(a, ocispec.Manifest{}), "aaa should be skipped")
	assert.False(t, hook.BeforePullLayer(b, ocispec.Manifest{}), "bbb should not be skipped")

	hook.AfterPullLayer(a, true, nil)
	wantErr := errors.New("network down")
	hook.AfterPullLayer(b, false, wantErr)

	assert.Equal(t, []ocispec.Descriptor{a, b}, hook.beforeCalls)
	assert.Equal(t, []afterCall{
		{desc: a, skipped: true, err: nil},
		{desc: b, skipped: false, err: wantErr},
	}, hook.afterCalls)
}

func TestNewFetch_DefaultHooks(t *testing.T) {
	f := NewFetch()
	assert.NotNil(t, f.Hooks)
	desc := ocispec.Descriptor{Digest: "sha256:cafe"}
	assert.False(t, f.Hooks.BeforePullLayer(desc, ocispec.Manifest{}))
	f.Hooks.AfterPullLayer(desc, false, nil)
}
