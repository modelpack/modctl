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
	"testing"
)

func TestNewLogin(t *testing.T) {
	login := NewLogin()
	if login.Username != "" {
		t.Errorf("expected empty username, got %s", login.Username)
	}
	if login.Password != "" {
		t.Errorf("expected empty password, got %s", login.Password)
	}
	if login.PasswordStdin != true {
		t.Errorf("expected PasswordStdin to be true, got %v", login.PasswordStdin)
	}
}

func TestLogin_Validate(t *testing.T) {
	tests := []struct {
		name    string
		login   *Login
		wantErr bool
		errMsg  string
	}{
		{
			name: "missing username",
			login: &Login{
				Username: "",
				Password: "password",
			},
			wantErr: true,
			errMsg:  "missing username",
		},
		{
			name: "missing password",
			login: &Login{
				Username: "username",
				Password: "",
			},
			wantErr: true,
			errMsg:  "missing password",
		},
		{
			name: "valid login",
			login: &Login{
				Username: "username",
				Password: "password",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.login.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && err.Error() != tt.errMsg {
				t.Errorf("Validate() error message = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}
