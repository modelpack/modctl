/*
 *     Copyright 2025 The CNAI Authors
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

package mlflow

import (
	"context"
	"errors"
	"net/url"
	"os"
	"slices"
	"strings"

	log "github.com/sirupsen/logrus"
)

// MlflowProvider implements the modelprovider.Provider interface for Mlflow
type MlflowProvider struct {
	mflClient MlFlowClient
}

// New creates a new ModelScope provider instance
func New() *MlflowProvider {
	return &MlflowProvider{}
}

// Name returns the name of this provider
func (p *MlflowProvider) Name() string {
	return "mlflow"
}

// SupportsURL checks if this provider can handle the given URL
// Supports experiments url (runs), model-registry(models)
// For short-form repo identifiers (owner/repo), users must explicitly specify --provider mlflow
func (p *MlflowProvider) SupportsURL(url string) bool {
	url = strings.TrimSpace(url)
	// TODO Mlflow API equals with Databricks Model Registry, support later
	possibleUrls := []string{"models:/"}

	return hasAnyPrefix(url, possibleUrls)
}

// DownloadModel downloads a model from ModelScope using the modelscope CLI
func (p *MlflowProvider) DownloadModel(
	ctx context.Context,
	modelURL, destDir string,
) (string, error) {
	model, version, err := parseModelURL(modelURL)
	if err != nil {
		return "", errors.Join(errors.New("Parse modelURL error"), err)
	}
	registryClient, err := NewMlFlowRegistry(nil)
	if err != nil {
		return "", errors.Join(errors.New("failed initialized mlflow registry client"), err)
	}
	downloadPath, err := registryClient.PullModelByName(ctx, model, version, destDir)
	if err != nil {
		return "", errors.Join(errors.New("provider download error"), err)
	}
	return downloadPath, nil
}

// CheckAuth verifies that the user is authenticated with MlFlow
func (p *MlflowProvider) CheckAuth() error {
	return checkMlflowAuth()
}

func hasAnyPrefix(s string, subs []string) bool {
	for _, sub := range subs {
		if strings.HasPrefix(s, sub) { // Check if the main string contains the current substring
			return true
		}
	}
	return false
}

func checkMlflowAuth() error {

	isAllNonEmpty := func(s []string) bool {
		for v := range slices.Values(s) {
			if v == "" {
				return false
			}
		}
		return true
	}

	databricksEnvs := []string{
		os.Getenv("DATABRICKS_HOST"),
		os.Getenv("DATABRICKS_USERNAME"),
		os.Getenv("DATABRICKS_PASSWORD"),
	}
	mlflowEnvs := []string{
		os.Getenv("MLFLOW_TRACKING_URI"),
		os.Getenv("MLFLOW_TRACKING_USERNAME"),
		os.Getenv("MLFLOW_TRACKING_PASSWORD"),
	}

	if isAllNonEmpty(databricksEnvs) {
		return nil
	} else if isAllNonEmpty(mlflowEnvs) {
		log.Printf("Detected MlFlow environment variables, set DATABRICKS_* envs \n")
	} else {
		log.Println("Please set DATABRICKS_HOST or MLFLOW_TRACKING_URI environment variable.")
		log.Println("Authentication for MLflow/Databricks is not configured.")
		log.Println("See https://pkg.go.dev/github.com/databricks/databricks-sdk-go/config for more details on configuration.")

		return errors.New("mlflow/databricks authentication not configured")
	}

	return nil
}

func parseModelURL(modelURL string) (string, string, error) {
	if modelURL == "" {
		return "", "", errors.New("modelUrl value missing.")
	}

	if strings.HasPrefix(modelURL, "models:") {
		parse, err := url.Parse(modelURL)
		if err != nil {
			return "", "", err
		}

		if parse == nil {
			return "", "", errors.New("model url is nil")
		}

		return parse.Hostname(), strings.TrimLeft(parse.Path, "/"), nil

	} else if strings.Contains(modelURL, "/") {

		split := strings.Split(modelURL, "/")

		if len(split) != 2 {
			return "", "", errors.New("model url is invalid, valid mask name/version")
		}
		return split[0], split[1], nil

	} else {
		return modelURL, "", nil
	}
}
