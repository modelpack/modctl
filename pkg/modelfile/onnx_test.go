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
	return encodeBytesField(onnxModelProtoGraphField, buildONNXGraphInitializerBytes(externalLocations))
}

// buildONNXGraphInitializerBytes returns the inner GraphProto payload — used
// directly when wrapping into a subgraph attribute or wherever raw graph bytes
// are needed without the outer ModelProto.graph header.
func buildONNXGraphInitializerBytes(externalLocations []string) []byte {
	var graph []byte
	for _, loc := range externalLocations {
		entry := encodeStringStringEntry(onnxExternalDataLocationKey, loc)
		tensor := encodeRepeated(onnxTensorProtoExternalDataField, entry)
		graph = append(graph, encodeBytesField(onnxGraphProtoInitializerField, tensor)...)
	}
	return graph
}

// buildONNXBytesWithConstantNodeAttribute constructs ONNX bytes where an
// external_data tensor is attached to a NodeProto.attribute.t (the shape
// produced by ONNX Constant ops). Pre-P2 the parser would have missed this.
func buildONNXBytesWithConstantNodeAttribute(location string) []byte {
	entry := encodeStringStringEntry(onnxExternalDataLocationKey, location)
	tensor := encodeRepeated(onnxTensorProtoExternalDataField, entry)
	attr := encodeBytesField(onnxAttributeProtoTensorField, tensor)
	node := encodeBytesField(onnxNodeProtoAttributeField, attr)
	graph := encodeBytesField(onnxGraphProtoNodeField, node)
	return encodeBytesField(onnxModelProtoGraphField, graph)
}

// buildONNXBytesWithSubgraph nests a GraphProto inside a NodeProto.attribute.g
// (the shape produced by ONNX If / Loop / Scan ops). Pre-P2 the parser would
// have missed external_data references inside the subgraph.
func buildONNXBytesWithSubgraph(location string) []byte {
	inner := buildONNXGraphInitializerBytes([]string{location})
	attr := encodeBytesField(onnxAttributeProtoGraphField, inner)
	node := encodeBytesField(onnxNodeProtoAttributeField, attr)
	outer := encodeBytesField(onnxGraphProtoNodeField, node)
	return encodeBytesField(onnxModelProtoGraphField, outer)
}

// buildONNXBytesWithTrainingInfo emits a ModelProto whose only graph-bearing
// fields are training_info[*].initialization and training_info[*].algorithm.
// The inference graph is intentionally absent — exercises the post-P4
// behavior where training-only ONNX still yields its external tensors.
func buildONNXBytesWithTrainingInfo(initLoc, algoLoc string) []byte {
	initGraph := buildONNXGraphInitializerBytes([]string{initLoc})
	algoGraph := buildONNXGraphInitializerBytes([]string{algoLoc})

	var ti []byte
	ti = append(ti, encodeBytesField(onnxTrainingInfoInitializationField, initGraph)...)
	ti = append(ti, encodeBytesField(onnxTrainingInfoAlgorithmField, algoGraph)...)

	return encodeBytesField(onnxModelProtoTrainingInfoField, ti)
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

// P2 coverage: external_data attached to a NodeProto.attribute.t (Constant op)
// must be discovered alongside top-level GraphProto.initializer entries.
func TestExtractONNXExternalDataPaths_NodeAttributeTensor(t *testing.T) {
	dir := t.TempDir()
	onnxPath := filepath.Join(dir, "model.onnx")
	require.NoError(t, os.WriteFile(onnxPath, buildONNXBytesWithConstantNodeAttribute("const_weights.bin"), 0o644))

	paths, err := ExtractONNXExternalDataPaths(onnxPath)
	require.NoError(t, err)
	assert.Equal(t, []string{"const_weights.bin"}, paths)
}

// P2 coverage: external_data inside a subgraph (If / Loop / Scan branch) must
// be discovered via attribute.g recursion.
func TestExtractONNXExternalDataPaths_Subgraph(t *testing.T) {
	dir := t.TempDir()
	onnxPath := filepath.Join(dir, "model.onnx")
	require.NoError(t, os.WriteFile(onnxPath, buildONNXBytesWithSubgraph("branch_weights.bin"), 0o644))

	paths, err := ExtractONNXExternalDataPaths(onnxPath)
	require.NoError(t, err)
	assert.Equal(t, []string{"branch_weights.bin"}, paths)
}

// P4 coverage: training-enabled ONNX (IR v7+) carries graphs in
// ModelProto.training_info[*].initialization and .algorithm. external_data in
// either of these must be discovered alongside the inference graph.
func TestExtractONNXExternalDataPaths_TrainingInfoGraphs(t *testing.T) {
	dir := t.TempDir()
	onnxPath := filepath.Join(dir, "model.onnx")
	require.NoError(t, os.WriteFile(
		onnxPath,
		buildONNXBytesWithTrainingInfo("init_state.bin", "optimizer_state.bin"),
		0o644,
	))

	paths, err := ExtractONNXExternalDataPaths(onnxPath)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"init_state.bin", "optimizer_state.bin"}, paths)
}

// P4 coverage: a model with both an inference graph AND training_info must
// surface external_data from both locations.
func TestExtractONNXExternalDataPaths_InferenceAndTrainingCombined(t *testing.T) {
	dir := t.TempDir()
	onnxPath := filepath.Join(dir, "model.onnx")

	// Inference graph initializer.
	inferenceGraph := buildONNXGraphInitializerBytes([]string{"weights.bin"})
	// training_info with one initialization graph.
	initGraph := buildONNXGraphInitializerBytes([]string{"init_state.bin"})
	ti := encodeBytesField(onnxTrainingInfoInitializationField, initGraph)

	// Compose ModelProto with both fields.
	var model []byte
	model = append(model, encodeBytesField(onnxModelProtoGraphField, inferenceGraph)...)
	model = append(model, encodeBytesField(onnxModelProtoTrainingInfoField, ti)...)
	require.NoError(t, os.WriteFile(onnxPath, model, 0o644))

	paths, err := ExtractONNXExternalDataPaths(onnxPath)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"weights.bin", "init_state.bin"}, paths)
}
