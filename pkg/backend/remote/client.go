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

package remote

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"

	"github.com/sirupsen/logrus"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/credentials"
	"oras.land/oras-go/v2/registry/remote/retry"

	"github.com/modelpack/modctl/pkg/version"
)

type Repository = remote.Repository

type Option func(*client)

type client struct {
	retry     bool
	plainHTTP bool
	insecure  bool
	proxy     string
}

func New(repo string, opts ...Option) (*remote.Repository, error) {
	client := &client{}
	for _, opt := range opts {
		opt(client)
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: client.insecure,
		},
	}

	if client.proxy != "" {
		proxyURL, err := url.Parse(client.proxy)
		if err != nil {
			return nil, fmt.Errorf("failed to parse the proxy URL: %w", err)
		}

		transport.Proxy = http.ProxyURL(proxyURL)
	}

	httpClient := &http.Client{}
	if client.retry {
		httpClient.Transport = retry.NewTransport(transport)
	} else {
		httpClient.Transport = transport
	}

	repository, err := remote.NewRepository(repo)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}

	// Load credentials from Docker config.
	credStore, err := credentials.NewStoreFromDocker(credentials.StoreOptions{AllowPlaintextPut: true})
	if err != nil {
		return nil, fmt.Errorf("failed to create credential store: %w", err)
	}

	repository.Client = &auth.Client{
		Cache:      auth.NewCache(),
		Credential: credentials.Credential(credStore),
		Client:     httpClient,
		Header:     makeHeader(),
	}

	repository.PlainHTTP = client.plainHTTP
	return repository, nil
}

func WithRetry(retry bool) Option {
	return func(c *client) {
		c.retry = retry
	}
}

func WithProxy(proxy string) Option {
	return func(c *client) {
		c.proxy = proxy
	}
}

func WithInsecure(insecure bool) Option {
	return func(c *client) {
		c.insecure = insecure
	}
}

func WithPlainHTTP(plainHTTP bool) Option {
	return func(c *client) {
		c.plainHTTP = plainHTTP
	}
}

// makeHeader creates a new http.Header with default headers.
func makeHeader() http.Header {
	header := make(http.Header)
	header.Set("User-Agent", fmt.Sprintf("modctl/%s", version.GitVersion))

	hostname, err := os.Hostname()
	if err != nil {
		logrus.Errorf("failed to get hostname: %v", err)
	} else {
		header.Set("X-Hostname", hostname)
	}

	ipAddr := getLocalIP()
	if ipAddr == "" {
		logrus.Errorf("failed to get local IP address")
	} else {
		header.Set("X-Host-Ip", ipAddr)
	}

	header.Set("X-Cpu-Arch", runtime.GOARCH)
	return header
}

// getLocalIP gets the local IP address.
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String()
			}
		}
	}

	return ""
}
