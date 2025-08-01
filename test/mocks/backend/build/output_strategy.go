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

// Code generated by mockery v2.53.3. DO NOT EDIT.

package build

import (
	context "context"
	io "io"

	hooks "github.com/modelpack/modctl/pkg/backend/build/hooks"

	mock "github.com/stretchr/testify/mock"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

// OutputStrategy is an autogenerated mock type for the OutputStrategy type
type OutputStrategy struct {
	mock.Mock
}

type OutputStrategy_Expecter struct {
	mock *mock.Mock
}

func (_m *OutputStrategy) EXPECT() *OutputStrategy_Expecter {
	return &OutputStrategy_Expecter{mock: &_m.Mock}
}

// OutputConfig provides a mock function with given fields: ctx, mediaType, digest, size, reader, _a5
func (_m *OutputStrategy) OutputConfig(ctx context.Context, mediaType string, digest string, size int64, reader io.Reader, _a5 hooks.Hooks) (v1.Descriptor, error) {
	ret := _m.Called(ctx, mediaType, digest, size, reader, _a5)

	if len(ret) == 0 {
		panic("no return value specified for OutputConfig")
	}

	var r0 v1.Descriptor
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, int64, io.Reader, hooks.Hooks) (v1.Descriptor, error)); ok {
		return rf(ctx, mediaType, digest, size, reader, _a5)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string, int64, io.Reader, hooks.Hooks) v1.Descriptor); ok {
		r0 = rf(ctx, mediaType, digest, size, reader, _a5)
	} else {
		r0 = ret.Get(0).(v1.Descriptor)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string, int64, io.Reader, hooks.Hooks) error); ok {
		r1 = rf(ctx, mediaType, digest, size, reader, _a5)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// OutputStrategy_OutputConfig_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'OutputConfig'
type OutputStrategy_OutputConfig_Call struct {
	*mock.Call
}

// OutputConfig is a helper method to define mock.On call
//   - ctx context.Context
//   - mediaType string
//   - digest string
//   - size int64
//   - reader io.Reader
//   - _a5 hooks.Hooks
func (_e *OutputStrategy_Expecter) OutputConfig(ctx interface{}, mediaType interface{}, digest interface{}, size interface{}, reader interface{}, _a5 interface{}) *OutputStrategy_OutputConfig_Call {
	return &OutputStrategy_OutputConfig_Call{Call: _e.mock.On("OutputConfig", ctx, mediaType, digest, size, reader, _a5)}
}

func (_c *OutputStrategy_OutputConfig_Call) Run(run func(ctx context.Context, mediaType string, digest string, size int64, reader io.Reader, _a5 hooks.Hooks)) *OutputStrategy_OutputConfig_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string), args[3].(int64), args[4].(io.Reader), args[5].(hooks.Hooks))
	})
	return _c
}

func (_c *OutputStrategy_OutputConfig_Call) Return(_a0 v1.Descriptor, _a1 error) *OutputStrategy_OutputConfig_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *OutputStrategy_OutputConfig_Call) RunAndReturn(run func(context.Context, string, string, int64, io.Reader, hooks.Hooks) (v1.Descriptor, error)) *OutputStrategy_OutputConfig_Call {
	_c.Call.Return(run)
	return _c
}

// OutputLayer provides a mock function with given fields: ctx, mediaType, relPath, digest, size, reader, _a6
func (_m *OutputStrategy) OutputLayer(ctx context.Context, mediaType string, relPath string, digest string, size int64, reader io.Reader, _a6 hooks.Hooks) (v1.Descriptor, error) {
	ret := _m.Called(ctx, mediaType, relPath, digest, size, reader, _a6)

	if len(ret) == 0 {
		panic("no return value specified for OutputLayer")
	}

	var r0 v1.Descriptor
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string, int64, io.Reader, hooks.Hooks) (v1.Descriptor, error)); ok {
		return rf(ctx, mediaType, relPath, digest, size, reader, _a6)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string, int64, io.Reader, hooks.Hooks) v1.Descriptor); ok {
		r0 = rf(ctx, mediaType, relPath, digest, size, reader, _a6)
	} else {
		r0 = ret.Get(0).(v1.Descriptor)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string, string, int64, io.Reader, hooks.Hooks) error); ok {
		r1 = rf(ctx, mediaType, relPath, digest, size, reader, _a6)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// OutputStrategy_OutputLayer_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'OutputLayer'
type OutputStrategy_OutputLayer_Call struct {
	*mock.Call
}

// OutputLayer is a helper method to define mock.On call
//   - ctx context.Context
//   - mediaType string
//   - relPath string
//   - digest string
//   - size int64
//   - reader io.Reader
//   - _a6 hooks.Hooks
func (_e *OutputStrategy_Expecter) OutputLayer(ctx interface{}, mediaType interface{}, relPath interface{}, digest interface{}, size interface{}, reader interface{}, _a6 interface{}) *OutputStrategy_OutputLayer_Call {
	return &OutputStrategy_OutputLayer_Call{Call: _e.mock.On("OutputLayer", ctx, mediaType, relPath, digest, size, reader, _a6)}
}

func (_c *OutputStrategy_OutputLayer_Call) Run(run func(ctx context.Context, mediaType string, relPath string, digest string, size int64, reader io.Reader, _a6 hooks.Hooks)) *OutputStrategy_OutputLayer_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string), args[3].(string), args[4].(int64), args[5].(io.Reader), args[6].(hooks.Hooks))
	})
	return _c
}

func (_c *OutputStrategy_OutputLayer_Call) Return(_a0 v1.Descriptor, _a1 error) *OutputStrategy_OutputLayer_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *OutputStrategy_OutputLayer_Call) RunAndReturn(run func(context.Context, string, string, string, int64, io.Reader, hooks.Hooks) (v1.Descriptor, error)) *OutputStrategy_OutputLayer_Call {
	_c.Call.Return(run)
	return _c
}

// OutputManifest provides a mock function with given fields: ctx, mediaType, digest, size, reader, _a5
func (_m *OutputStrategy) OutputManifest(ctx context.Context, mediaType string, digest string, size int64, reader io.Reader, _a5 hooks.Hooks) (v1.Descriptor, error) {
	ret := _m.Called(ctx, mediaType, digest, size, reader, _a5)

	if len(ret) == 0 {
		panic("no return value specified for OutputManifest")
	}

	var r0 v1.Descriptor
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, int64, io.Reader, hooks.Hooks) (v1.Descriptor, error)); ok {
		return rf(ctx, mediaType, digest, size, reader, _a5)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string, int64, io.Reader, hooks.Hooks) v1.Descriptor); ok {
		r0 = rf(ctx, mediaType, digest, size, reader, _a5)
	} else {
		r0 = ret.Get(0).(v1.Descriptor)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string, int64, io.Reader, hooks.Hooks) error); ok {
		r1 = rf(ctx, mediaType, digest, size, reader, _a5)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// OutputStrategy_OutputManifest_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'OutputManifest'
type OutputStrategy_OutputManifest_Call struct {
	*mock.Call
}

// OutputManifest is a helper method to define mock.On call
//   - ctx context.Context
//   - mediaType string
//   - digest string
//   - size int64
//   - reader io.Reader
//   - _a5 hooks.Hooks
func (_e *OutputStrategy_Expecter) OutputManifest(ctx interface{}, mediaType interface{}, digest interface{}, size interface{}, reader interface{}, _a5 interface{}) *OutputStrategy_OutputManifest_Call {
	return &OutputStrategy_OutputManifest_Call{Call: _e.mock.On("OutputManifest", ctx, mediaType, digest, size, reader, _a5)}
}

func (_c *OutputStrategy_OutputManifest_Call) Run(run func(ctx context.Context, mediaType string, digest string, size int64, reader io.Reader, _a5 hooks.Hooks)) *OutputStrategy_OutputManifest_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].(string), args[3].(int64), args[4].(io.Reader), args[5].(hooks.Hooks))
	})
	return _c
}

func (_c *OutputStrategy_OutputManifest_Call) Return(_a0 v1.Descriptor, _a1 error) *OutputStrategy_OutputManifest_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *OutputStrategy_OutputManifest_Call) RunAndReturn(run func(context.Context, string, string, int64, io.Reader, hooks.Hooks) (v1.Descriptor, error)) *OutputStrategy_OutputManifest_Call {
	_c.Call.Return(run)
	return _c
}

// NewOutputStrategy creates a new instance of OutputStrategy. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewOutputStrategy(t interface {
	mock.TestingT
	Cleanup(func())
}) *OutputStrategy {
	mock := &OutputStrategy{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
