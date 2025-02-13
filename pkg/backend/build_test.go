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

	"github.com/CloudNativeAI/modctl/test/mocks/modelfile"

	"github.com/stretchr/testify/assert"
)

func TestGetProcessors(t *testing.T) {
	modelfile := &modelfile.Modelfile{}
	modelfile.On("GetConfigs").Return([]string{"config1", "config2"})
	modelfile.On("GetModels").Return([]string{"model1", "model2"})

	processors := getProcessors(modelfile)

	assert.Len(t, processors, 4)
	assert.Equal(t, "license", processors[0].Name())
	assert.Equal(t, "readme", processors[1].Name())
	assert.Equal(t, "model_config", processors[2].Name())
	assert.Equal(t, "model", processors[3].Name())
}
