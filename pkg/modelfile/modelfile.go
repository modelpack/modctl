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

package modelfile

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"

	modefilecommand "github.com/CloudNativeAI/modctl/pkg/modelfile/command"
	"github.com/CloudNativeAI/modctl/pkg/modelfile/parser"
	"github.com/emirpasic/gods/sets/hashset"
)

// Modelfile is the interface for the modelfile. It is used to parse
// the modelfile by the path and get the information of the modelfile.
type Modelfile interface {
	// GetConfigs returns the args of the config command in the modelfile,
	// and deduplicates the args. The order of the args is the same as the
	// order in the modelfile.
	GetConfigs() []string

	// GetModels returns the args of the model command in the modelfile,
	// and deduplicates the args. The order of the args is the same as The
	// order in the modelfile.
	GetModels() []string

	// GetName returns the value of the name command in the modelfile.
	GetName() string

	// GetArch returns the value of the arch command in the modelfile.
	GetArch() string

	// GetFamily returns the value of the family command in the modelfile.
	GetFamily() string

	// GetFormat returns the value of the format command in the modelfile.
	GetFormat() string

	// GetParamsize returns the value of the paramsize command in the modelfile.
	GetParamsize() string

	// GetPrecision returns the value of the precision command in the modelfile.
	GetPrecision() string

	// GetQuantization returns the value of the quantization command in the modelfile.
	GetQuantization() string
}

// modelfile is the implementation of the Modelfile interface.
type modelfile struct {
	config       *hashset.Set
	model        *hashset.Set
	name         string
	arch         string
	family       string
	format       string
	paramsize    string
	precision    string
	quantization string
}

// NewModelfile creates a new modelfile by the path of the modelfile.
// It parses the modelfile and returns the modelfile interface.
func NewModelfile(path string) (Modelfile, error) {
	mf := &modelfile{
		config: hashset.New(),
		model:  hashset.New(),
	}
	if err := mf.parseFile(path); err != nil {
		return nil, err
	}

	return mf, nil
}

// parseFile parses the modelfile by the path, and validates the args of the commands.
func (mf *modelfile) parseFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	ast, err := parser.Parse(f)
	if err != nil {
		return err
	}

	for _, child := range ast.GetChildren() {
		switch child.GetValue() {
		case modefilecommand.CONFIG:
			mf.config.Add(child.GetNext().GetValue())
		case modefilecommand.MODEL:
			mf.model.Add(child.GetNext().GetValue())
		case modefilecommand.NAME:
			if mf.name != "" {
				return fmt.Errorf("duplicate name command on line %d", child.GetStartLine())
			}
			mf.name = child.GetNext().GetValue()
		case modefilecommand.ARCH:
			if mf.arch != "" {
				return fmt.Errorf("duplicate arc command on line %d", child.GetStartLine())
			}
			mf.arch = child.GetNext().GetValue()
		case modefilecommand.FAMILY:
			if mf.family != "" {
				return fmt.Errorf("duplicate family command on line %d", child.GetStartLine())
			}
			mf.family = child.GetNext().GetValue()
		case modefilecommand.FORMAT:
			if mf.format != "" {
				return fmt.Errorf("duplicate format command on line %d", child.GetStartLine())
			}
			mf.format = child.GetNext().GetValue()
		case modefilecommand.PARAMSIZE:
			if mf.paramsize != "" {
				return fmt.Errorf("duplicate paramsize command on line %d", child.GetStartLine())
			}
			mf.paramsize = child.GetNext().GetValue()
		case modefilecommand.PRECISION:
			if mf.precision != "" {
				return fmt.Errorf("duplicate precision command on line %d", child.GetStartLine())
			}
			mf.precision = child.GetNext().GetValue()
		case modefilecommand.QUANTIZATION:
			if mf.quantization != "" {
				return fmt.Errorf("duplicate quantization command on line %d", child.GetStartLine())
			}
			mf.quantization = child.GetNext().GetValue()
		default:
			return fmt.Errorf("unknown command %s on line %d", child.GetValue(), child.GetStartLine())
		}
	}

	return nil
}

// GetConfigs returns the args of the config command in the modelfile,
// and deduplicates the args. The order of the args is the same as the
// order in the modelfile.
func (mf *modelfile) GetConfigs() []string {
	var configs []string
	for _, rawConfig := range mf.config.Values() {
		config, ok := rawConfig.(string)
		if !ok {
			log.Warnf("failed to convert config to string: %v", rawConfig)
			continue
		}

		configs = append(configs, config)
	}

	return configs
}

// GetModels returns the args of the model command in the modelfile,
// and deduplicates the args. The order of the args is the same as the
// order in the modelfile.
func (mf *modelfile) GetModels() []string {
	var models []string
	for _, rawModel := range mf.model.Values() {
		model, ok := rawModel.(string)
		if !ok {
			log.Warnf("failed to convert model to string: %v", rawModel)
			continue
		}

		models = append(models, model)
	}

	return models
}

// GetName returns the value of the name command in the modelfile.
func (mf *modelfile) GetName() string {
	return mf.name
}

// GetArch returns the value of the arch command in the modelfile.
func (mf *modelfile) GetArch() string {
	return mf.arch
}

// GetFamily returns the value of the family command in the modelfile.
func (mf *modelfile) GetFamily() string {
	return mf.family
}

// GetFormat returns the value of the format command in the modelfile.
func (mf *modelfile) GetFormat() string {
	return mf.format
}

// GetParamsize returns the value of the paramsize command in the modelfile.
func (mf *modelfile) GetParamsize() string {
	return mf.paramsize
}

// GetPrecision returns the value of the precision command in the modelfile.
func (mf *modelfile) GetPrecision() string {
	return mf.precision
}

// GetQuantization returns the value of the quantization command in the modelfile.
func (mf *modelfile) GetQuantization() string {
	return mf.quantization
}
