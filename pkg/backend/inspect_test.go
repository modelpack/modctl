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

package backend

import (
	"context"
	"testing"

	"github.com/CloudNativeAI/modctl/test/mocks/storage"
	"github.com/stretchr/testify/assert"
)

func TestInspect(t *testing.T) {
	mockStore := &storage.Storage{}
	b := &backend{store: mockStore}
	ctx := context.Background()
	target := "example.com/repo:tag"
	manifest := `{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.manifest.v1+json",
  "artifactType": "application/vnd.cnai.model.manifest.v1+json",
  "config": {
    "mediaType": "application/vnd.oci.image.config.v1+json",
    "digest": "sha256:144ac462bafbbc7cc6c9e6b325049a0aca1b6ffa2f6cfb0a80ec64bc690bec04",
    "size": 46
  },
  "layers": [
    {
      "mediaType": "application/vnd.oci.image.layer.v1.tar",
      "digest": "sha256:5a96686deb327903f4310e9181ef2ee0bc7261e5181bd23ccdce6c575b6120a2",
      "size": 13312,
      "annotations": {
        "org.cnai.model.filepath": "LICENSE",
        "org.cnai.model.license": "true"
      },
      "artifactType": "application/vnd.cnai.model.layer.v1.tar"
    },
    {
      "mediaType": "application/vnd.oci.image.layer.v1.tar",
      "digest": "sha256:44a6e989cc7084ef35aedf1dd7090204ccc928829c51ce79d7d59c346a228333",
      "size": 5632,
      "annotations": {
        "org.cnai.model.filepath": "README.md",
        "org.cnai.model.readme": "true"
      },
      "artifactType": "application/vnd.cnai.model.layer.v1.tar"
    },
    {
      "mediaType": "application/vnd.oci.image.layer.v1.tar",
      "digest": "sha256:a4e7c313c8addcc5f8ac3d87d48a9af7eb89bf8819c869c9fa0cad1026397b0c",
      "size": 2560,
      "annotations": {
        "org.cnai.model.config": "true",
        "org.cnai.model.filepath": "config.json"
      },
      "artifactType": "application/vnd.cnai.model.layer.v1.tar"
    },
    {
      "mediaType": "application/vnd.oci.image.layer.v1.tar",
      "digest": "sha256:567f11b7338855adbaf58c8e195455860400ef148fc7f02ebc446efdb8b7c515",
      "size": 1536,
      "annotations": {
        "org.cnai.model.config": "true",
        "org.cnai.model.filepath": "foo/bar.json"
      },
      "artifactType": "application/vnd.cnai.model.layer.v1.tar"
    },
    {
      "mediaType": "application/vnd.oci.image.layer.v1.tar",
      "digest": "sha256:628ce381719b65598622e3f71844192f84e135d937c7b5a8116582edbe3b1f5d",
      "size": 2048,
      "annotations": {
        "org.cnai.model.config": "true",
        "org.cnai.model.filepath": "generation_config.json"
      },
      "artifactType": "application/vnd.cnai.model.layer.v1.tar"
    },
    {
      "mediaType": "application/vnd.oci.image.layer.v1.tar",
      "digest": "sha256:0480097912f4dd530382c69f00d41409bc51f62ea146a04d70c0254791f4ac32",
      "size": 7033344,
      "annotations": {
        "org.cnai.model.config": "true",
        "org.cnai.model.filepath": "tokenizer.json"
      },
      "artifactType": "application/vnd.cnai.model.layer.v1.tar"
    },
    {
      "mediaType": "application/vnd.oci.image.layer.v1.tar",
      "digest": "sha256:ebea935e6c2de57780addfc0262c30c2f83afb1457a124fd9b22370e6cb5bc34",
      "size": 9216,
      "annotations": {
        "org.cnai.model.config": "true",
        "org.cnai.model.filepath": "tokenizer_config.json"
      },
      "artifactType": "application/vnd.cnai.model.layer.v1.tar"
    },
    {
      "mediaType": "application/vnd.oci.image.layer.v1.tar",
      "digest": "sha256:3a2844a891e19d1d183ac12918a497116309ba9abe0523cdcf1874cf8aebe8e0",
      "size": 2778624,
      "annotations": {
        "org.cnai.model.config": "true",
        "org.cnai.model.filepath": "vocab.json"
      },
      "artifactType": "application/vnd.cnai.model.layer.v1.tar"
    }
  ],
  "annotations": {
    "org.cnai.model.architecture": "transformer",
    "org.cnai.model.created": "2024-11-11T21:16:41+08:00",
    "org.cnai.model.family": "qwen2",
    "org.cnai.model.format": "tensorflow",
    "org.cnai.model.name": "Qwen2.5-0.5B",
    "org.cnai.model.param.size": "0.49B",
    "org.cnai.model.precision": "int8",
    "org.cnai.model.quantization": "gptq"
  }
}`

	mockStore.On("PullManifest", ctx, "example.com/repo", "tag").Return([]byte(manifest), "sha256:2bc8836f5910ec63a01109e20db67c2ad7706cb19bef5a303bc86fa5572ec9a2", nil)

	inspected, err := b.Inspect(ctx, target)
	assert.NoError(t, err)
	assert.Equal(t, "sha256:144ac462bafbbc7cc6c9e6b325049a0aca1b6ffa2f6cfb0a80ec64bc690bec04", inspected.ID)
	assert.Equal(t, "sha256:2bc8836f5910ec63a01109e20db67c2ad7706cb19bef5a303bc86fa5572ec9a2", inspected.Digest)
	assert.Equal(t, "transformer", inspected.Architecture)
	assert.Equal(t, "2024-11-11T21:16:41+08:00", inspected.Created)
	assert.Equal(t, "qwen2", inspected.Family)
	assert.Equal(t, "tensorflow", inspected.Format)
	assert.Equal(t, "Qwen2.5-0.5B", inspected.Name)
	assert.Equal(t, "0.49B", inspected.ParamSize)
	assert.Equal(t, "int8", inspected.Precision)
	assert.Equal(t, "gptq", inspected.Quantization)
	assert.Len(t, inspected.Layers, 8)
	assert.Equal(t, "sha256:5a96686deb327903f4310e9181ef2ee0bc7261e5181bd23ccdce6c575b6120a2", inspected.Layers[0].Digest)
	assert.Equal(t, "LICENSE", inspected.Layers[0].Filepath)
	assert.Equal(t, int64(13312), inspected.Layers[0].Size)
}
