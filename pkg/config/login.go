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

import "fmt"

type Login struct {
	Username      string
	Password      string
	PasswordStdin bool
	AuthFilePath  string
	PlainHTTP     bool
	Insecure      bool
}

// AuthConfigEntry holds authentication credentials for a registry.
type AuthConfigEntry struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Auth     string `json:"auth,omitempty"`
}

// AuthConfig is a structure for dockerconfigjson‑style files.
type AuthConfig struct {
	Auths map[string]AuthConfigEntry `json:"auths"`
}

func NewLogin() *Login {
	return &Login{
		Username:      "",
		Password:      "",
		PasswordStdin: true,
		AuthFilePath:  "",
		PlainHTTP:     false,
		Insecure:      false,
	}
}

func (l *Login) Validate() error {
	if len(l.AuthFilePath) != 0 {
		if len(l.Username) != 0 || len(l.Password) != 0 {
			return fmt.Errorf("--authfile cannot be used with --username or --password")
		}
		return nil
	}

	if len(l.Username) == 0 {
		return fmt.Errorf("missing username")
	}

	if len(l.Password) == 0 && !l.PasswordStdin {
		return fmt.Errorf("missing password")
	}

	return nil
}
