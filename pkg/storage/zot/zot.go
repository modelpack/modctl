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

package zot

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"sync"

	godigest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"zotregistry.dev/zot/pkg/api/config"
	"zotregistry.dev/zot/pkg/log"
	zstorage "zotregistry.dev/zot/pkg/storage"
	types "zotregistry.dev/zot/pkg/storage/types"
)

const (
	// StorageTypeZot is the storage type of zot.
	StorageTypeZot = "zot"
)

var (
	// globalStorage is the global storage instance implemented by zot.
	globalStorage *storage
	// once useds to ensure the global storage is initialized only once.
	once sync.Once
)

// storage is a storage implementation based on the storage of project zot.
type storage struct {
	// imageStore is the image store of zot.
	imageStore types.ImageStore
}

// NewStorage creates a new storage.
func NewStorage(rootDir string) (*storage, error) {
	var err error
	if globalStorage == nil {
		once.Do(func() {
			// make sure the root content directory exists.
			if err = ensureDir(rootDir); err != nil {
				err = fmt.Errorf("failed to ensure the directory %s: %w", rootDir, err)
				return
			}
			// create the image store.
			cfg := &config.Config{
				Storage: config.GlobalStorageConfig{
					StorageConfig: config.StorageConfig{
						RootDirectory: rootDir,
						Dedupe:        true,
					},
				},
			}

			var sc zstorage.StoreController
			sc, err = zstorage.New(cfg, nil, &nopMetricsServer{}, log.NewLogger("disabled", ""))
			if err != nil {
				err = fmt.Errorf("failed to create the image store: %w", err)
				return
			}

			globalStorage = &storage{imageStore: sc.GetDefaultImageStore()}
		})
	}

	return globalStorage, err
}

// GetIndex gets the content of index.json in repo from the storage.
func (s *storage) GetIndex(ctx context.Context, repo string) ([]byte, error) {
	return s.imageStore.GetIndexContent(repo)
}

// PullManifest pulls the manifest from the storage.
func (s *storage) PullManifest(ctx context.Context, repo, reference string) ([]byte, string, error) {
	manifest, digest, _, err := s.imageStore.GetImageManifest(repo, reference)
	return manifest, digest.String(), err
}

// PushManifest pushes the manifest to the storage.
func (s *storage) PushManifest(ctx context.Context, repo, reference string, body []byte) (string, error) {
	digest, _, err := s.imageStore.PutImageManifest(repo, reference, ocispec.MediaTypeImageManifest, body)
	if err != nil {
		return "", err
	}

	return digest.String(), nil
}

// DeleteManifest deletes the manifest from the storage.
func (s *storage) DeleteManifest(ctx context.Context, repo, reference string) error {
	return s.imageStore.DeleteImageManifest(repo, reference, true)
}

// PushBlob pushes the blob to the storage.
func (s *storage) PushBlob(ctx context.Context, repo string, body io.Reader) (string, int64, error) {
	// push the blob should be splited into three steps:
	// 1. request for upload to reteive the upload location(uuid).
	// 2. upload the blob by stream to the location.
	// 3. commit the upload to finish the upload process.
	uuid, err := s.imageStore.NewBlobUpload(repo)
	if err != nil {
		return "", 0, err
	}

	// wrap the body reader to teeReader used to calculate the digest.
	hash := sha256.New()
	body = io.TeeReader(body, hash)
	size, err := s.imageStore.PutBlobChunkStreamed(repo, uuid, body)
	if err != nil {
		return "", 0, err
	}

	// commit the upload.
	digest := godigest.NewDigest(godigest.SHA256, hash)
	if err := s.imageStore.FinishBlobUpload(repo, uuid, body, digest); err != nil {
		return "", 0, err
	}

	return digest.String(), size, nil
}

// StatBlob stats the blob in the storage.
func (s *storage) StatBlob(ctx context.Context, repo, digest string) (bool, error) {
	exist, _, _, err := s.imageStore.StatBlob(repo, godigest.Digest(digest))
	return exist, err
}

// DeleteBlob deletes the blob from the storage.
func (s *storage) DeleteBlob(ctx context.Context, repo, digest string) error {
	return s.imageStore.DeleteBlob(repo, godigest.Digest(digest))
}

// ListBlobs lists all the blobs in the repository.
func (s *storage) ListBlobs(ctx context.Context, repo string) ([]string, error) {
	digests, err := s.imageStore.GetAllBlobs(repo)
	if err != nil {
		return nil, err
	}

	digestsStr := make([]string, 0, len(digests))
	for _, d := range digests {
		digestsStr = append(digestsStr, d.String())
	}

	return digestsStr, nil
}

// ListRepositories lists the repositories in the storage.
func (s *storage) ListRepositories(ctx context.Context) ([]string, error) {
	return s.imageStore.GetRepositories()
}

// CleanupRepo cleans up the repository in the storage.
func (s *storage) CleanupRepo(ctx context.Context, repo string, blobs []string, removeRepo bool) (int, error) {
	digests := make([]godigest.Digest, 0, len(blobs))
	for _, b := range blobs {
		digests = append(digests, godigest.Digest(b))
	}

	return s.imageStore.CleanupRepo(repo, digests, removeRepo)
}

// ListTags lists the tags of the repository in the storage.
func (s *storage) ListTags(ctx context.Context, repo string) ([]string, error) {
	return s.imageStore.GetImageTags(repo)
}

// ensureDir ensures the directory exists.
func ensureDir(path string) error {
	return os.MkdirAll(path, 0755)
}
