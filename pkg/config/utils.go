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

package config

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

func ParseAuthFile(path, registry string) (string, string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("read authfile: %w", err)
	}
	var cfg AuthConfig
	if err = json.Unmarshal(b, &cfg); err != nil {
		return "", "", fmt.Errorf("decode json: %w", err)
	}
	return ExtractCred(cfg, registry)
}

func ExtractCred(cfg AuthConfig, registry string) (user, pass string, err error) {
	entry, ok := cfg.Auths[registry]
	if !ok {
		return "", "", fmt.Errorf("registry %q not found in authfile", registry)
	}

	switch {
	case entry.Username != "" && entry.Password != "":
		return entry.Username, entry.Password, nil
	case entry.Auth != "":
		decoded, err := base64.StdEncoding.DecodeString(entry.Auth)
		if err != nil {
			return "", "", fmt.Errorf("base64 decode: %w", err)
		}
		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) != 2 {
			return "", "", errors.New("malformed auth (expected username:password)")
		}
		return parts[0], parts[1], nil
	default:
		return "", "", errors.New("no username/password or auth field present for registry")
	}
}
