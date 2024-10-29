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

package spec

const (
	// AnnotationCreated is the annotation key for the date and time on which the model was built (date-time string as defined by RFC 3339).
	AnnotationCreated = "org.cnai.model.created"

	// AnnotationArchitecture is the annotation key for the model architecture, such as `transformer`, `cnn`, `rnn`, etc.
	AnnotationArchitecture = "org.cnai.model.architecture"

	// AnnotationFamily is the annotation key for the model family, such as `llama3`, `gpt2`, `qwen2`, etc.
	AnnotationFamily = "org.cnai.model.family"

	// AnnotationName is the annotation key for the model name, such as `llama3-8b-instruct`, `gpt2-xl`, `qwen2-vl-72b-instruct`, etc.
	AnnotationName = "org.cnai.model.name"

	// AnnotationFormat is the annotation key for the model format, such as `onnx`, `tensorflow`, `pytorch`, etc.
	AnnotationFormat = "org.cnai.model.format"

	// AnnotationParamSize is the annotation key for the size of the model parameters.
	AnnotationParamSize = "org.cnai.model.param.size"

	// AnnotationPrecision is the annotation key for the model precision, such as `bf16`, `fp16`, `int8`, etc.
	AnnotationPrecision = "org.cnai.model.precision"

	// AnnotationQuantization is the annotation key for the model quantization, such as `awq`, `gptq`, etc.
	AnnotationQuantization = "org.cnai.model.quantization"
)

const (
	// ArtifactTypeModelManifest specifies the media type for a model manifest.
	ArtifactTypeModelManifest = "application/vnd.cnai.model.manifest.v1+json"
)

const (
	// ArtifactTypeModelLayer is the media type used for layers referenced by the manifest.
	ArtifactTypeModelLayer = "application/vnd.cnai.model.layer.v1.tar"

	// ArtifactTypeModelLayerGzip is the media type used for gzipped layers
	// referenced by the manifest.
	ArtifactTypeModelLayerGzip = "application/vnd.cnai.model.layer.v1.tar+gzip"
)

const (
	// AnnotationReadme is the annotation key for the layer is a README.md file (boolean), such as `true` or `false`.
	AnnotationReadme = "org.cnai.model.readme"

	// AnnotationLicense is the annotation key for the layer is a LICENSE file (boolean), such as `true` or `false`.
	AnnotationLicense = "org.cnai.model.license"

	// AnnotationConfig is the annotation key for the layer is a config file (boolean), such as `true` or `false`.
	AnnotationConfig = "org.cnai.model.config"

	// AnnotationModel is the annotation key for the layer is a model file (boolean), such as `true` or `false`.
	AnnotationModel = "org.cnai.model.model"
)
