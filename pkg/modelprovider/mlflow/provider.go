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
	"fmt"
	"net/url"
	"os"
	"strings"
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
	possibleUrls := []string{"models"}

	return hasAnyPrefix(url, possibleUrls)
}

// DownloadModel downloads a model from ModelScope using the modelscope CLI
func (p *MlflowProvider) DownloadModel(ctx context.Context, modelURL, destDir string) (string, error) {
	model, version, err := parseModelURL(modelURL)
	if err != nil {
		return "", err
	}
	registryClient, err := NewMlFlowRegistry(nil)
	if err != nil {
		return "", err
	}
	downloadPath, err := registryClient.PullModelByName(ctx, model, version, destDir)
	if err != nil {
		return "", err
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

	var err error

	host := os.Getenv("DATABRICKS_HOST")
	usr := os.Getenv("DATABRICKS_USERNAME")
	pass := os.Getenv("DATABRICKS_PASSWORD")
	mlfhost := os.Getenv("MLFLOW_TRACKING_URI")
	mlfuser := os.Getenv("MLFLOW_TRACKING_USERNAME")
	mlfpass := os.Getenv("MLFLOW_TRACKING_PASSWORD")

	if host == "" && usr == "" && pass == "" {
		fmt.Println("Please set DATABRICKS_HOST environment variable.")
		fmt.Println("Please set DATABRICKS_USERNAME environment variable.")
		fmt.Println("Please set DATABRICKS_PASSWORD environment variable.")
	} else {
		return nil
	}
	if mlfhost != "" && mlfuser != "" && mlfpass != "" {
		err = os.Setenv("DATABRICKS_HOST", mlfhost)
		if err != nil {
			return err
		}
		err = os.Setenv("DATABRICKS_USERNAME", usr)
		if err != nil {
			return err
		}
		err = os.Setenv("DATABRICKS_PASSWORD", pass)
		if err != nil {
			return err
		}

	} else {
		return errors.New("please set MLFLOW tracking environment variable.")
	}
	return err
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
