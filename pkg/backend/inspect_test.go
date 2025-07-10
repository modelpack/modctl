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
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	pkgconfig "github.com/CloudNativeAI/modctl/pkg/config"
	"github.com/CloudNativeAI/modctl/test/mocks/storage"
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
    "mediaType": "application/vnd.cnai.model.config.v1+json",
    "digest": "sha256:e31b55920173ba79526491fbd01efe609c1d0d72c3a83df85b2c4fe74df2eea2",
    "size": 277
  },
  "layers": [
    {
      "mediaType": "application/vnd.cnai.model.doc.v1.tar",
      "digest": "sha256:5a96686deb327903f4310e9181ef2ee0bc7261e5181bd23ccdce6c575b6120a2",
      "size": 13312,
      "annotations": {
        "org.cnai.model.filepath": "LICENSE"
      }
    },
    {
      "mediaType": "application/vnd.cnai.model.doc.v1.tar",
      "digest": "sha256:44a6e989cc7084ef35aedf1dd7090204ccc928829c51ce79d7d59c346a228333",
      "size": 5632,
      "annotations": {
        "org.cnai.model.filepath": "README.md"
      }
    },
    {
      "mediaType": "application/vnd.cnai.model.weight.config.v1.tar",
      "digest": "sha256:a4e7c313c8addcc5f8ac3d87d48a9af7eb89bf8819c869c9fa0cad1026397b0c",
      "size": 2560,
      "annotations": {
        "org.cnai.model.filepath": "config.json"
      }
    },
    {
      "mediaType": "application/vnd.cnai.model.weight.config.v1.tar",
      "digest": "sha256:567f11b7338855adbaf58c8e195455860400ef148fc7f02ebc446efdb8b7c515",
      "size": 1536,
      "annotations": {
        "org.cnai.model.filepath": "foo/bar.json"
      }
    },
    {
      "mediaType": "application/vnd.cnai.model.weight.config.v1.tar",
      "digest": "sha256:628ce381719b65598622e3f71844192f84e135d937c7b5a8116582edbe3b1f5d",
      "size": 2048,
      "annotations": {
        "org.cnai.model.filepath": "generation_config.json"
      }
    },
    {
      "mediaType": "application/vnd.cnai.model.weight.config.v1.tar",
      "digest": "sha256:0480097912f4dd530382c69f00d41409bc51f62ea146a04d70c0254791f4ac32",
      "size": 7033344,
      "annotations": {
        "org.cnai.model.filepath": "tokenizer.json"
      }
    },
    {
      "mediaType": "application/vnd.cnai.model.weight.config.v1.tar",
      "digest": "sha256:ebea935e6c2de57780addfc0262c30c2f83afb1457a124fd9b22370e6cb5bc34",
      "size": 9216,
      "annotations": {
        "org.cnai.model.filepath": "tokenizer_config.json"
      }
    },
    {
      "mediaType": "application/vnd.cnai.model.weight.config.v1.tar",
      "digest": "sha256:3a2844a891e19d1d183ac12918a497116309ba9abe0523cdcf1874cf8aebe8e0",
      "size": 2778624,
      "annotations": {
        "org.cnai.model.filepath": "vocab.json"
      }
    }
  ]
}`
	config := `{
  "descriptor": {
    "createdAt": "2025-02-12T17:01:43.968027+08:00",
    "family": "qwen2",
    "name": "Qwen2.5-0.5B"
  },
  "modelfs": {
    "type": "layers",
    "diff_ids": null
  },
  "config": {
    "architecture": "transformer",
    "format": "tensorflow",
    "paramSize": "0.5b",
    "precision": "int8",
    "quantization": "gptq"
  }
}`

	mockStore.On("PullManifest", ctx, "example.com/repo", "tag").Return([]byte(manifest), "sha256:9ca701e8784e5656e2c36f10f82410a0af4c44f859590a28a3d1519ee1eea89d", nil)
	mockStore.On("PullBlob", ctx, "example.com/repo", "sha256:e31b55920173ba79526491fbd01efe609c1d0d72c3a83df85b2c4fe74df2eea2").Return(io.NopCloser(bytes.NewReader([]byte(config))), nil)

	inspectedAny, err := b.Inspect(ctx, target, &pkgconfig.Inspect{})
	inspected := inspectedAny.(*InspectedModelArtifact)
	assert.NoError(t, err)
	assert.Equal(t, "sha256:e31b55920173ba79526491fbd01efe609c1d0d72c3a83df85b2c4fe74df2eea2", inspected.ID)
	assert.Equal(t, "sha256:9ca701e8784e5656e2c36f10f82410a0af4c44f859590a28a3d1519ee1eea89d", inspected.Digest)
	assert.Equal(t, "transformer", inspected.Architecture)
	assert.Equal(t, "2025-02-12T17:01:43+08:00", inspected.CreatedAt)
	assert.Equal(t, "qwen2", inspected.Family)
	assert.Equal(t, "tensorflow", inspected.Format)
	assert.Equal(t, "Qwen2.5-0.5B", inspected.Name)
	assert.Equal(t, "0.5b", inspected.ParamSize)
	assert.Equal(t, "int8", inspected.Precision)
	assert.Equal(t, "gptq", inspected.Quantization)
	assert.Len(t, inspected.Layers, 8)
	assert.Equal(t, "application/vnd.cnai.model.doc.v1.tar", inspected.Layers[0].MediaType)
	assert.Equal(t, "sha256:5a96686deb327903f4310e9181ef2ee0bc7261e5181bd23ccdce6c575b6120a2", inspected.Layers[0].Digest)
	assert.Equal(t, "LICENSE", inspected.Layers[0].Filepath)
	assert.Equal(t, int64(13312), inspected.Layers[0].Size)
}
