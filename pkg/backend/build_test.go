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
	"testing"

	"github.com/CloudNativeAI/modctl/pkg/config"
	"github.com/CloudNativeAI/modctl/test/mocks/modelfile"

	"github.com/stretchr/testify/assert"
)

func TestGetProcessors(t *testing.T) {
	modelfile := &modelfile.Modelfile{}
	modelfile.On("GetConfigs").Return([]string{"config1", "config2"})
	modelfile.On("GetModels").Return([]string{"model1", "model2"})
	modelfile.On("GetModelFlags").Return(make(map[string]map[string]string))
	modelfile.On("GetCodes").Return([]string{"1.py", "2.py"})
	modelfile.On("GetCodeFlags").Return(make(map[string]map[string]string))
	modelfile.On("GetDocs").Return([]string{"doc1", "doc2"})

	b := &backend{}
	processors := b.getProcessors(modelfile, &config.Build{})

	assert.Len(t, processors, 4)
	assert.Equal(t, "config", processors[0].Name())
	assert.Equal(t, "model", processors[1].Name())
	assert.Equal(t, "code", processors[2].Name())
	assert.Equal(t, "doc", processors[3].Name())
}
