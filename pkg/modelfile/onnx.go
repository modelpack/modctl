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
	onnxModelProtoGraphField              = 7
	onnxGraphProtoInitializerField        = 5
	onnxGraphProtoSparseInitializerField  = 15
	onnxTensorProtoExternalDataField      = 13
	onnxSparseTensorProtoValuesField      = 1
	onnxSparseTensorProtoIndicesField     = 2
	onnxStringStringEntryProtoKeyField    = 1
	onnxStringStringEntryProtoValueField  = 2
	onnxExternalDataLocationKey           = "location"
	onnxMaxParseSize                  int = 512 * 1024 * 1024
)

// ExtractONNXExternalDataPaths parses the ONNX model file at onnxPath and returns
// the relative paths of all external_data tensor files referenced by it.
//
// Paths are returned exactly as recorded in the ONNX file (relative to the
// directory containing the .onnx file, per ONNX convention). The slice is
// deduplicated and ordered by first appearance.
//
// Returns nil with no error for files that aren't ONNX or that don't reference
// any external data; only I/O failures produce errors. Files larger than
// onnxMaxParseSize are skipped with an error.
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

	graph, err := readSubMessage(data, onnxModelProtoGraphField)
	if err != nil {
		return nil, fmt.Errorf("locate ONNX graph field: %w", err)
	}
	if graph == nil {
		return nil, nil
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

	if err := walkGraph(graph, collect); err != nil {
		return nil, fmt.Errorf("walk ONNX graph: %w", err)
	}
	return locations, nil
}

// walkGraph iterates initializer + sparse_initializer entries inside a GraphProto.
func walkGraph(graph []byte, collect func(string)) error {
	return forEachField(graph, func(num protowire.Number, _ protowire.Type, value []byte) error {
		switch num {
		case onnxGraphProtoInitializerField:
			return walkTensorExternalData(value, collect)
		case onnxGraphProtoSparseInitializerField:
			return walkSparseTensor(value, collect)
		}
		return nil
	})
}

// walkSparseTensor descends into a SparseTensorProto and walks its values + indices TensorProto.
func walkSparseTensor(sparse []byte, collect func(string)) error {
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

// readSubMessage scans the top-level message and returns the bytes of the first
// occurrence of the given length-delimited (wire type 2) field. Returns nil if
// the field is absent.
func readSubMessage(buf []byte, target protowire.Number) ([]byte, error) {
	var found []byte
	err := forEachField(buf, func(num protowire.Number, typ protowire.Type, value []byte) error {
		if num == target && typ == protowire.BytesType && found == nil {
			found = value
		}
		return nil
	})
	return found, err
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
