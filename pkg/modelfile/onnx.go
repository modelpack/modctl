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
	"fmt"
	"os"

	"google.golang.org/protobuf/encoding/protowire"
)

// ONNX protobuf field numbers (subset relevant to external_data discovery).
// Schema reference: https://github.com/onnx/onnx/blob/main/onnx/onnx.proto
const (
	onnxModelProtoGraphField             = 7
	onnxModelProtoTrainingInfoField      = 20
	onnxTrainingInfoInitializationField  = 1
	onnxTrainingInfoAlgorithmField       = 2
	onnxGraphProtoNodeField              = 1
	onnxGraphProtoInitializerField       = 5
	onnxGraphProtoSparseInitializerField = 15
	onnxNodeProtoAttributeField          = 5
	// AttributeProto carries either a tensor (t / tensors / sparse_tensor /
	// sparse_tensors) or a subgraph (g / graphs). Field numbers per onnx.proto.
	onnxAttributeProtoTensorField        = 5
	onnxAttributeProtoGraphField         = 6
	onnxAttributeProtoTensorsField       = 10
	onnxAttributeProtoGraphsField        = 11
	onnxAttributeProtoSparseTensorField  = 22
	onnxAttributeProtoSparseTensorsField = 23
	onnxTensorProtoExternalDataField     = 13
	onnxSparseTensorProtoValuesField     = 1
	onnxSparseTensorProtoIndicesField    = 2
	onnxStringStringEntryProtoKeyField   = 1
	onnxStringStringEntryProtoValueField = 2
	onnxExternalDataLocationKey          = "location"
	onnxMaxParseSize                 int = 512 * 1024 * 1024
	// Defends against pathological / adversarial ONNX with deeply nested If/Loop
	// subgraphs. Real models rarely nest beyond 2-3 levels.
	onnxMaxSubgraphDepth = 32
)

// ExtractONNXExternalDataPaths parses the ONNX model file at onnxPath and returns
// the relative paths of all external_data tensor files referenced by it.
//
// Paths are returned exactly as recorded in the ONNX file (relative to the
// directory containing the .onnx file, per ONNX convention). The slice is
// deduplicated and ordered by first appearance.
//
// An ONNX without any external_data references returns nil, nil. The function
// returns an error for: I/O failures (stat / read), files exceeding
// onnxMaxParseSize, and malformed protobuf wire data (e.g., truncated or
// corrupted bytes). Callers that want best-effort behavior should treat all
// errors as a fallback signal and continue without external_data information.
func ExtractONNXExternalDataPaths(onnxPath string) ([]string, error) {
	info, err := os.Stat(onnxPath)
	if err != nil {
		return nil, fmt.Errorf("stat onnx file: %w", err)
	}
	if info.Size() > int64(onnxMaxParseSize) {
		return nil, fmt.Errorf("onnx file %s exceeds parse size cap (%d bytes)", onnxPath, onnxMaxParseSize)
	}

	data, err := os.ReadFile(onnxPath)
	if err != nil {
		return nil, fmt.Errorf("read onnx file: %w", err)
	}

	var (
		seen      = map[string]struct{}{}
		locations []string
	)
	collect := func(loc string) {
		if loc == "" {
			return
		}
		if _, dup := seen[loc]; dup {
			return
		}
		seen[loc] = struct{}{}
		locations = append(locations, loc)
	}

	// Iterate ModelProto fields directly so we cover both the inference graph
	// (field 7) and any training_info entries (field 20, repeated). Training-
	// enabled ONNX (IR v7+) carries additional GraphProtos in
	// training_info[*].initialization and training_info[*].algorithm; their
	// initializers can also carry external_data references.
	err = forEachField(data, func(num protowire.Number, _ protowire.Type, value []byte) error {
		switch num {
		case onnxModelProtoGraphField:
			return walkGraph(value, collect, 0)
		case onnxModelProtoTrainingInfoField:
			return walkTrainingInfo(value, collect)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk ONNX model: %w", err)
	}
	return locations, nil
}

// walkTrainingInfo descends into a TrainingInfoProto and walks its
// initialization and algorithm GraphProtos. The other fields
// (initialization_binding, update_binding) cannot carry external_data.
func walkTrainingInfo(ti []byte, collect func(string)) error {
	return forEachField(ti, func(num protowire.Number, _ protowire.Type, value []byte) error {
		switch num {
		case onnxTrainingInfoInitializationField, onnxTrainingInfoAlgorithmField:
			return walkGraph(value, collect, 0)
		}
		return nil
	})
}

// walkGraph iterates a GraphProto: top-level initializer / sparse_initializer
// entries plus every NodeProto.attribute, recursing into subgraphs (If / Loop /
// Scan branches) up to onnxMaxSubgraphDepth levels deep.
func walkGraph(graph []byte, collect func(string), depth int) error {
	if depth > onnxMaxSubgraphDepth {
		return nil
	}
	return forEachField(graph, func(num protowire.Number, _ protowire.Type, value []byte) error {
		switch num {
		case onnxGraphProtoInitializerField:
			return walkTensorExternalData(value, collect)
		case onnxGraphProtoSparseInitializerField:
			return walkSparseTensor(value, collect, depth)
		case onnxGraphProtoNodeField:
			return walkNode(value, collect, depth)
		}
		return nil
	})
}

// walkNode descends into NodeProto.attribute entries to surface external_data
// references attached via Constant / If / Loop / Scan and similar ops.
func walkNode(node []byte, collect func(string), depth int) error {
	return forEachField(node, func(num protowire.Number, _ protowire.Type, value []byte) error {
		if num != onnxNodeProtoAttributeField {
			return nil
		}
		return walkAttribute(value, collect, depth)
	})
}

// walkAttribute handles the tensor- and subgraph-bearing fields of
// AttributeProto. Other attribute types (floats / ints / strings / type_protos)
// are skipped — none can carry external_data.
func walkAttribute(attr []byte, collect func(string), depth int) error {
	return forEachField(attr, func(num protowire.Number, _ protowire.Type, value []byte) error {
		switch num {
		case onnxAttributeProtoTensorField, onnxAttributeProtoTensorsField:
			return walkTensorExternalData(value, collect)
		case onnxAttributeProtoSparseTensorField, onnxAttributeProtoSparseTensorsField:
			return walkSparseTensor(value, collect, depth)
		case onnxAttributeProtoGraphField, onnxAttributeProtoGraphsField:
			return walkGraph(value, collect, depth+1)
		}
		return nil
	})
}

// walkSparseTensor descends into a SparseTensorProto and walks its values + indices TensorProto.
func walkSparseTensor(sparse []byte, collect func(string), _ int) error {
	return forEachField(sparse, func(num protowire.Number, _ protowire.Type, value []byte) error {
		switch num {
		case onnxSparseTensorProtoValuesField, onnxSparseTensorProtoIndicesField:
			return walkTensorExternalData(value, collect)
		}
		return nil
	})
}

// walkTensorExternalData scans a TensorProto for external_data StringStringEntryProto entries.
func walkTensorExternalData(tensor []byte, collect func(string)) error {
	return forEachField(tensor, func(num protowire.Number, _ protowire.Type, value []byte) error {
		if num != onnxTensorProtoExternalDataField {
			return nil
		}
		var key, val string
		if err := forEachField(value, func(n protowire.Number, _ protowire.Type, v []byte) error {
			switch n {
			case onnxStringStringEntryProtoKeyField:
				key = string(v)
			case onnxStringStringEntryProtoValueField:
				val = string(v)
			}
			return nil
		}); err != nil {
			return err
		}
		if key == onnxExternalDataLocationKey {
			collect(val)
		}
		return nil
	})
}

// forEachField iterates protobuf wire-format fields in buf, invoking fn for each.
// For length-delimited fields, value is the inner payload bytes (header stripped).
// For varint/fixed32/fixed64/group fields, value is nil and the caller can rely on
// the field number/type to decide whether to consume further. Unknown/skipped
// fields advance the cursor without error.
func forEachField(buf []byte, fn func(num protowire.Number, typ protowire.Type, value []byte) error) error {
	for len(buf) > 0 {
		num, typ, n := protowire.ConsumeTag(buf)
		if err := protowire.ParseError(n); err != nil {
			return err
		}
		buf = buf[n:]

		switch typ {
		case protowire.BytesType:
			value, m := protowire.ConsumeBytes(buf)
			if err := protowire.ParseError(m); err != nil {
				return err
			}
			if err := fn(num, typ, value); err != nil {
				return err
			}
			buf = buf[m:]
		default:
			m := protowire.ConsumeFieldValue(num, typ, buf)
			if err := protowire.ParseError(m); err != nil {
				return err
			}
			if err := fn(num, typ, nil); err != nil {
				return err
			}
			buf = buf[m:]
		}
	}
	return nil
}
