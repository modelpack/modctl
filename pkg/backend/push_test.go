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

package backend_test

import (
	"context"
	"errors"
	"testing"

	mocks "github.com/CloudNativeAI/modctl/test/mocks/backend"

	"github.com/stretchr/testify/assert"
)

func TestPush(t *testing.T) {
	ctx := context.Background()
	target1 := "example.com/test-repo:should_error"
	target2 := "example.com/test-repo:should_not_error"

	b := &mocks.Backend{}
	b.On("Push", ctx, target1).Return(errors.New("mock error"))
	err := b.Push(ctx, target1)
	assert.Error(t, err, "Push should return an error")

	b.On("Push", ctx, target2).Return(nil)
	err = b.Push(ctx, target2)
	assert.NoError(t, err, "Push should not return an error")
}
