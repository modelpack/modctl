/*
 *     Copyright 2025 The ModelPack Authors
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

package modelfile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
)

// buildMinimalONNXBytes constructs a wire-format byte sequence matching the
// minimal subtree of ONNX needed to exercise external_data extraction:
//
//   ModelProto {
//     graph: GraphProto {
//       initializer: TensorProto[i] {
//         external_data: StringStringEntryProto[]{ key="location", value=<path> }
//       }
//     }
//   }
//
// One initializer is emitted per externalLocation. Field numbers come from
// onnx.proto and are mirrored as constants in onnx.go.
func buildMinimalONNXBytes(externalLocations []string) []byte {
	var graph []byte
	for _, loc := range externalLocations {
		entry := encodeStringStringEntry(onnxExternalDataLocationKey, loc)
		tensor := encodeRepeated(onnxTensorProtoExternalDataField, entry)
		graph = append(graph, encodeBytesField(onnxGraphProtoInitializerField, tensor)...)
	}
	return encodeBytesField(onnxModelProtoGraphField, graph)
}

func encodeStringStringEntry(key, value string) []byte {
	out := encodeBytesField(onnxStringStringEntryProtoKeyField, []byte(key))
	out = append(out, encodeBytesField(onnxStringStringEntryProtoValueField, []byte(value))...)
	return out
}

// encodeRepeated wraps a single repeated-message instance so it appears as one
// entry of a repeated bytes-typed field.
func encodeRepeated(fieldNum int, payload []byte) []byte {
	return encodeBytesField(fieldNum, payload)
}

func encodeBytesField(fieldNum int, payload []byte) []byte {
	out := protowire.AppendTag(nil, protowire.Number(fieldNum), protowire.BytesType)
	out = protowire.AppendBytes(out, payload)
	return out
}

func TestExtractONNXExternalDataPaths_Empty(t *testing.T) {
	dir := t.TempDir()
	onnxPath := filepath.Join(dir, "model.onnx")
	require.NoError(t, os.WriteFile(onnxPath, buildMinimalONNXBytes(nil), 0o644))

	paths, err := ExtractONNXExternalDataPaths(onnxPath)
	require.NoError(t, err)
	assert.Empty(t, paths)
}

func TestExtractONNXExternalDataPaths_SingleExternal(t *testing.T) {
	dir := t.TempDir()
	onnxPath := filepath.Join(dir, "model.onnx")
	require.NoError(t, os.WriteFile(onnxPath, buildMinimalONNXBytes([]string{"weights.bin"}), 0o644))

	paths, err := ExtractONNXExternalDataPaths(onnxPath)
	require.NoError(t, err)
	assert.Equal(t, []string{"weights.bin"}, paths)
}

func TestExtractONNXExternalDataPaths_MultipleExternals(t *testing.T) {
	// Mimics the screenshot ONNX layout: a model.onnx referencing many tensor
	// files with arbitrary, extension-less names.
	locations := []string{
		"tower_deep_layer_0_kernel_read__448_1",
		"tower_shallow_layer_0_kernel_read__440_0",
		"moe_layer_layer_0_kernel__399_15",
		"feature_gate_main_kernel_read__352_27",
		"external_data_for_resource_handle",
	}
	dir := t.TempDir()
	onnxPath := filepath.Join(dir, "model.onnx")
	require.NoError(t, os.WriteFile(onnxPath, buildMinimalONNXBytes(locations), 0o644))

	paths, err := ExtractONNXExternalDataPaths(onnxPath)
	require.NoError(t, err)
	assert.Equal(t, locations, paths)
}

func TestExtractONNXExternalDataPaths_DeduplicatesRepeatedLocation(t *testing.T) {
	dir := t.TempDir()
	onnxPath := filepath.Join(dir, "model.onnx")
	require.NoError(t, os.WriteFile(onnxPath, buildMinimalONNXBytes([]string{"a", "b", "a", "c", "b"}), 0o644))

	paths, err := ExtractONNXExternalDataPaths(onnxPath)
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, paths)
}

func TestExtractONNXExternalDataPaths_FileMissing(t *testing.T) {
	_, err := ExtractONNXExternalDataPaths(filepath.Join(t.TempDir(), "nope.onnx"))
	assert.Error(t, err)
}

func TestExtractONNXExternalDataPaths_NonONNXBytes(t *testing.T) {
	dir := t.TempDir()
	onnxPath := filepath.Join(dir, "model.onnx")
	require.NoError(t, os.WriteFile(onnxPath, []byte("not a protobuf"), 0o644))

	// Garbage bytes are not valid wire format; expect either no paths or a
	// parse error. Both are acceptable — the contract is "don't panic, don't
	// silently return wrong data."
	paths, err := ExtractONNXExternalDataPaths(onnxPath)
	if err == nil {
		assert.Empty(t, paths)
	}
}
