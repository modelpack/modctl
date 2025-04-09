/*
 *     Copyright 2025 The CNAI Authors
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

package interceptor

import (
	"context"
	"io"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// ApplyDescriptorFn is a function that applies changes to the descriptor.
type ApplyDescriptorFn func(desc *ocispec.Descriptor)

// Interceptor is an interface that defines the interceptor for the building stream.
type Interceptor interface {
	// Intercept intercepts the building stream for some customized logic, readerType is the original stream type, such as raw or tar.
	Intercept(ctx context.Context, mediaType string, filepath string, readerType string, reader io.Reader) (ApplyDescriptorFn, error)
}
