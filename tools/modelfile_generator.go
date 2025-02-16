package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/CloudNativeAI/modctl/pkg/modelfile/command"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type ModelfileGenConfig struct {
	Version       string
	Name          string
	Arch          string
	Family        string
	Format        string
	Paramsize     int
	Precision     string
	Quantization  string
	OverwriteArgs bool
	OutputPath    string
}

var genConfig = &ModelfileGenConfig{}

// modelfileGenCmd represents the modctl tools command for generate modelfile.
var modelfileGenCmd = &cobra.Command{
	Use:                "genmodelfile [flags] <path>",
	Short:              "A command line tool for generating modelfile.",
	Args:               cobra.ExactArgs(1),
	DisableAutoGenTag:  true,
	SilenceUsage:       true,
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGenModelfile(context.Background(), args[0])
	},
}

// init initializes build command.
func init() {
	flags := modelfileGenCmd.Flags()

	flags.StringVarP(&genConfig.Version, "version", "v", "v1", "Model version (string), such as v1, v2, etc.")
	flags.StringVarP(&genConfig.Name, "name", "n", "", "Model name (string), such as llama3-8b-instruct, gpt2-xl, qwen2-vl-72b-instruct, etc.")
	flags.StringVar(&genConfig.Arch, "arch", "", "Model architecture (string), such as transformer, cnn, rnn, etc.")
	flags.StringVar(&genConfig.Family, "family", "", "Model family (string), such as llama3, gpt2, qwen2, etc.")
	flags.StringVar(&genConfig.Format, "format", "", "Model format (string), such as onnx, tensorflow, pytorch, etc.")
	flags.IntVar(&genConfig.Paramsize, "paramsize", 0, "Number of parameters in the model (integer).")
	flags.StringVar(&genConfig.Precision, "precision", "", "Model precision (string), such as bf16, fp16, int8, etc.")
	flags.StringVar(&genConfig.Quantization, "quantization", "", "Model quantization (string), such as awq, gptq, etc.")
	flags.BoolVar(&genConfig.OverwriteArgs, "overwrite_args", false, "Overwrite model arguments (boolean).")
	flags.StringVarP(&genConfig.OutputPath, "output_path", "o", "./", "Output path (string), such as /path/to/output.")

	if err := viper.BindPFlags(flags); err != nil {
		panic(fmt.Errorf("bind cache list flags to viper: %w", err))
	}
}

func runGenModelfile(ctx context.Context, path string) error {
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}

	if genConfig.Name == "" {
		genConfig.Name = filepath.Base(strings.TrimSuffix(path, "/"))
	}

	parameters := map[string]interface{}{
		command.NAME:         genConfig.Name,
		command.ARCH:         genConfig.Arch,
		command.FAMILY:       genConfig.Family,
		command.FORMAT:       genConfig.Format,
		command.PARAMSIZE:    genConfig.Paramsize,
		command.PRECISION:    genConfig.Precision,
		command.QUANTIZATION: genConfig.Quantization,
	}

	err := parseModelConfig(path, parameters, genConfig.OverwriteArgs)
	if err != nil {
		panic(fmt.Errorf("Error parsing model config: %v", err))
	}

	modelfile := ""
	for k, v := range parameters {
		fmt.Printf("%s %v\n", k, v)
		if k == "name" || k == "arch" || k == "family" || k == "format" || k == "paramsize" || k == "precision" || k == "quantization" {
			if v != nil && v != "" {
				modelfile += fmt.Sprintf("%s %v\n\n", strings.ToUpper(k), v)
			}
		}
	}

	modelfile += "\n# Model weight, support regex.\n"
	modelFiles := getFiles(path, []string{"onnx", "pt", "bin", "safetensors"})
	for suffix, count := range modelFiles {
		if count > 0 {
			modelfile += fmt.Sprintf("MODEL \\.%s$\n", suffix)
		}
	}

	modelfile += "\n# Specify model configuration file, support regex.\n"
	configFiles := getFiles(path, []string{"LICENSE", "json", "jsonl", "yaml", "yml", "txt", "config", "modelcard", "meta", "py", "md"})
	for suffix, count := range configFiles {
		if count > 0 {
			modelfile += fmt.Sprintf("CONFIG \\.%s$\n", suffix)
		}
	}

	if _, err := os.Stat(genConfig.OutputPath); os.IsNotExist(err) {
		fmt.Println("Output path %s does not exist. Creating it now.\n", genConfig.OutputPath)
		if err := os.MkdirAll(genConfig.OutputPath, 0755); err != nil {
			panic(fmt.Errorf("Error creating output path: %v", err))
		}
	}

	outputFile := filepath.Join(genConfig.OutputPath, "Modelfile")
	if err := ioutil.WriteFile(outputFile, []byte(modelfile), 0644); err != nil {
		panic(fmt.Errorf("Error writing Modelfile: %v", err))
	}
	return nil
}

func getFiles(path string, suffixLists []string) map[string]int {
	files := make(map[string]int)
	for _, suffix := range suffixLists {
		files[suffix] = 0
	}

	err := filepath.Walk(path, func(fullPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		suffix := strings.ToLower(filepath.Ext(fullPath))[1:]
		if _, ok := files[suffix]; ok {
			files[suffix]++
		}
		return nil
	})
	if err != nil {
		panic(fmt.Errorf("Error walking path: %v", err))
	}
	return files
}

func parseModelConfig(path string, parameters map[string]interface{}, overwriteArgs bool) error {
	modelConfig := make(map[string]interface{})
	for _, file := range []string{"config.json", "generation_config.json"} {
		filename := filepath.Join(path, file)
		data, err := ioutil.ReadFile(filename)
		if err == nil {
			var config map[string]interface{}
			if err := json.Unmarshal(data, &config); err == nil {
				for k, v := range config {
					modelConfig[k] = v
				}
			}
		}
	}
	updateModelConfig(modelConfig, parameters, command.PRECISION, []string{"torch_dtype"}, overwriteArgs)
	updateModelConfig(modelConfig, parameters, command.FAMILY, []string{"model_type"}, overwriteArgs)

	if _, ok := modelConfig["transformers_version"]; ok || parameters[command.ARCH] == "transformer" {
		parameters[command.ARCH] = "transformer"
	}
	return nil
}

func updateModelConfig(config map[string]interface{}, parameters map[string]interface{}, key string, keyInConfig []string, overwriteArgs bool) {
	if parameters[key] != nil && !overwriteArgs {
		return
	}
	for _, keyInConfig := range keyInConfig {
		if value, ok := config[keyInConfig]; ok {
			parameters[key] = value
			fmt.Println("set %s to %v according to config file under model path", key, value)
			return
		}
	}
}
