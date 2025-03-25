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
	"context"
	"crypto/tls"
	"net/http"

	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
	"oras.land/oras-go/v2/registry/remote/retry"

	"github.com/CloudNativeAI/modctl/pkg/config"
)

// Login logs into a registry.
func (b *backend) Login(ctx context.Context, registry, username, password string, cfg *config.Login) error {
	// read credentials from docker store.
	store, err := credentials.NewStoreFromDocker(credentials.StoreOptions{AllowPlaintextPut: true})
	if err != nil {
		return err
	}

	reg, err := remote.NewRegistry(registry)
	if err != nil {
		return err
	}

	httpClient := &http.Client{
		Transport: retry.NewTransport(&http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: cfg.Insecure,
			},
		}),
	}
	reg.Client = &auth.Client{
		Cache:      auth.NewCache(),
		Credential: credentials.Credential(store),
		Client:     httpClient,
	}

	if cfg.PlainHTTP {
		reg.PlainHTTP = true
	}

	cred := auth.Credential{
		Username: username,
		Password: password,
	}

	return credentials.Login(ctx, store, reg, cred)
}
