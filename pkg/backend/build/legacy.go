/*
 *     Copyright 2026 The ModelPack Authors
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

package build

// Transitional wire-format identifiers (vnd.cnai.* / org.cnai.*).
//
// The source-of-truth model-spec moved from github.com/dragonflyoss/model-spec
// (vnd.cnai.* / org.cnai.*) to github.com/modelpack/model-spec (vnd.cncf.* /
// org.cncf.*) when the project graduated into the CNCF sandbox. Registries
// that registered their MODEL artifact processor against the legacy
// identifiers cannot dispatch vnd.cncf.* manifests and return
// "processor for artifact MODEL not found" on /additions endpoints.
//
// These constants pin the wire-format strings emitted by modctl to the
// legacy values for compatibility with such registries. Once downstream
// registries register processors for the modelpack identifiers, delete
// this file and switch the build-path call sites back to modelspec.*.
const (
	LegacyArtifactTypeModelManifest = "application/vnd.cnai.model.manifest.v1+json"

	LegacyMediaTypeModelConfig = "application/vnd.cnai.model.config.v1+json"

	LegacyMediaTypeModelWeight          = "application/vnd.cnai.model.weight.v1.tar"
	LegacyMediaTypeModelWeightRaw       = "application/vnd.cnai.model.weight.v1.raw"
	LegacyMediaTypeModelWeightConfig    = "application/vnd.cnai.model.weight.config.v1.tar"
	LegacyMediaTypeModelWeightConfigRaw = "application/vnd.cnai.model.weight.config.v1.raw"
	LegacyMediaTypeModelCode            = "application/vnd.cnai.model.code.v1.tar"
	LegacyMediaTypeModelCodeRaw         = "application/vnd.cnai.model.code.v1.raw"
	LegacyMediaTypeModelDoc             = "application/vnd.cnai.model.doc.v1.tar"
	LegacyMediaTypeModelDocRaw          = "application/vnd.cnai.model.doc.v1.raw"

	LegacyAnnotationFilepath     = "org.cnai.model.filepath"
	LegacyAnnotationFileMetadata = "org.cnai.model.file.metadata+json"
)
