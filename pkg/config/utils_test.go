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
	"os"
	"testing"
)

func TestExtractCred(t *testing.T) {
	b64 := base64.StdEncoding.EncodeToString([]byte("user2:pass2"))

	cases := []struct {
		name     string
		cfg      AuthConfig
		registry string
		wantUser string
		wantPass string
		wantErr  bool
	}{
		{
			name:     "username password fields",
			cfg:      AuthConfig{Auths: map[string]AuthConfigEntry{"example.io": {Username: "u", Password: "p"}}},
			registry: "example.io",
			wantUser: "u",
			wantPass: "p",
		},
		{
			name:     "auth base64 field",
			cfg:      AuthConfig{Auths: map[string]AuthConfigEntry{"registry.local": {Auth: b64}}},
			registry: "registry.local",
			wantUser: "user2",
			wantPass: "pass2",
		},
		{
			name:     "registry missing",
			cfg:      AuthConfig{Auths: map[string]AuthConfigEntry{}},
			registry: "miss.io",
			wantErr:  true,
		},
		{
			name:     "malformed auth",
			cfg:      AuthConfig{Auths: map[string]AuthConfigEntry{"bad": {Auth: base64.StdEncoding.EncodeToString([]byte("onlyuser"))}}},
			registry: "bad",
			wantErr:  true,
		},
		{
			name:     "empty entry",
			cfg:      AuthConfig{Auths: map[string]AuthConfigEntry{"empty": {}}},
			registry: "empty",
			wantErr:  true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			user, pass, err := ExtractCred(tc.cfg, tc.registry)
			if (err != nil) != tc.wantErr {
				t.Fatalf("expected err=%v got %v", tc.wantErr, err)
			}
			if !tc.wantErr {
				if user != tc.wantUser || pass != tc.wantPass {
					t.Fatalf("want (%s,%s) got (%s,%s)", tc.wantUser, tc.wantPass, user, pass)
				}
			}
		})
	}
}

// minimal I/O test to ensure ParseAuthFile ties together read+unmarshal+ExtractCred
func TestParseAuthFile(t *testing.T) {
	cfg := AuthConfig{Auths: map[string]AuthConfigEntry{"io": {Username: "a", Password: "b"}}}

	tmp, err := os.CreateTemp(t.TempDir(), "authfile-*.json")
	if err != nil {
		t.Fatalf("temp: %v", err)
	}
	defer tmp.Close()

	enc := json.NewEncoder(tmp)
	if err := enc.Encode(cfg); err != nil {
		t.Fatalf("encode: %v", err)
	}

	path := tmp.Name()

	user, pass, err := ParseAuthFile(path, "io")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if user != "a" || pass != "b" {
		t.Fatalf("want (a,b) got (%s,%s)", user, pass)
	}

	// ensure file read error propagates
	if _, _, err := ParseAuthFile("/non/exist", "io"); err == nil {
		t.Fatalf("expected error for missing file")
	}
}
