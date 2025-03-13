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

package distribution

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"

	distribution "github.com/distribution/distribution/v3"
	registry "github.com/distribution/distribution/v3/registry/storage"
	"github.com/distribution/distribution/v3/registry/storage/driver"
	"github.com/distribution/distribution/v3/registry/storage/driver/filesystem"
	ref "github.com/distribution/reference"
	sha256 "github.com/minio/sha256-simd"
	godigest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
)

func init() {
	// The PathRegexp in the distribution package is used to validate the repository name,
	// which not cover the case of the repository name includes the :port, so mutate the regexp to support it.
	// original regexp: ^(/[A-Za-z0-9._-]+)+$
	// patched regexp:  ^(/[A-Za-z0-9._:-]+)+$
	driver.PathRegexp = regexp.MustCompile(`^(/[A-Za-z0-9._:-]+)+$`)
}

const (
	// StorageTypeDistribution is the storage type of distribution.
	StorageTypeDistribution = "distribution"
	// defaultMaxThreads is the default max threads of the storage.
	defaultMaxThreads = 100
)

type storage struct {
	// driver is the underlying storage implementation.
	driver driver.StorageDriver
	// store represents a collection of repositories, addressable by name.
	store distribution.Namespace
}

func NewStorage(rootDir string) (*storage, error) {
	// Mute the logging from distribution.
	logrus.SetOutput(io.Discard)

	fsDriver := filesystem.New(filesystem.DriverParameters{
		RootDirectory: rootDir,
		MaxThreads:    defaultMaxThreads,
	})
	store, err := registry.NewRegistry(context.Background(), fsDriver)
	if err != nil {
		return nil, err
	}

	return &storage{driver: fsDriver, store: store}, nil
}

// repository gets the distribution repository service.
func (s *storage) repository(ctx context.Context, repo string) (distribution.Repository, error) {
	named, err := ref.ParseNamed(repo)
	if err != nil {
		return nil, err
	}

	return s.store.Repository(ctx, named)
}

// PullManifest pulls the manifest from the storage.
func (s *storage) PullManifest(ctx context.Context, repo, reference string) ([]byte, string, error) {
	repository, err := s.repository(ctx, repo)
	if err != nil {
		return nil, "", err
	}

	manifest, err := repository.Manifests(ctx)
	if err != nil {
		return nil, "", err
	}

	tag, err := repository.Tags(ctx).Get(ctx, reference)
	if err != nil {
		return nil, "", err
	}

	imageManifest, err := manifest.Get(ctx, tag.Digest)
	if err != nil {
		return nil, "", err
	}

	_, payload, err := imageManifest.Payload()
	if err != nil {
		return nil, "", err
	}

	return payload, tag.Digest.String(), nil
}

// PushManifest pushes the manifest to the storage.
func (s *storage) PushManifest(ctx context.Context, repo, reference string, manifestBytes []byte) (string, error) {
	repository, err := s.repository(ctx, repo)
	if err != nil {
		return "", err
	}

	manifest, err := repository.Manifests(ctx)
	if err != nil {
		return "", err
	}

	// TODO: pass in the mediatype from function parameters.
	imageManifest, desc, err := distribution.UnmarshalManifest(ocispec.MediaTypeImageManifest, manifestBytes)
	if err != nil {
		return "", err
	}

	digest, err := manifest.Put(ctx, imageManifest)
	if err != nil {
		return "", err
	}

	// tag the manifest.
	if err := repository.Tags(ctx).Tag(ctx, reference, desc); err != nil {
		return "", err
	}

	return digest.String(), nil
}

// DeleteManifest deletes the manifest from the storage.
func (s *storage) DeleteManifest(ctx context.Context, repo, reference string) error {
	repository, err := s.repository(ctx, repo)
	if err != nil {
		return err
	}

	// check whether the reference is a digest.
	digest, err := godigest.Parse(reference)
	if err == nil {
		// delete the manifest by digest.
		manifest, err := repository.Manifests(ctx)
		if err != nil {
			return err
		}

		return manifest.Delete(ctx, digest)
	} else {
		// only untagged the manifest if the reference is a tag.
		return repository.Tags(ctx).Untag(ctx, reference)
	}
}

// PullBlob pulls the blob from the storage.
func (s *storage) PullBlob(ctx context.Context, repo, digest string) (io.ReadCloser, error) {
	repository, err := s.repository(ctx, repo)
	if err != nil {
		return nil, err
	}

	return repository.Blobs(ctx).Open(ctx, godigest.Digest(digest))
}

// PushBlob pushes the blob to the storage.
func (s *storage) PushBlob(ctx context.Context, repo string, blobReader io.Reader, provisional ocispec.Descriptor) (string, int64, error) {
	repository, err := s.repository(ctx, repo)
	if err != nil {
		return "", 0, err
	}

	hash := sha256.New()
	if provisional.Digest == "" {
		blobReader = io.TeeReader(blobReader, hash)
	}

	blob, err := repository.Blobs(ctx).Create(ctx)
	if err != nil {
		return "", 0, err
	}

	size, err := blob.ReadFrom(blobReader)
	if err != nil {
		return "", 0, err
	}

	// if the provided provisional descriptor is not empty, we can just use it to commit,
	// otherwise we need to calculate the digest.
	if provisional.Digest == "" {
		provisional.Digest = godigest.Digest(fmt.Sprintf("sha256:%x", hash.Sum(nil)))
		provisional.Size = size
	}

	desc, err := blob.Commit(ctx, provisional)
	if err != nil {
		return "", 0, nil
	}

	return desc.Digest.String(), desc.Size, nil
}

// MountBlob mounts the blob to the storage.
func (s *storage) MountBlob(ctx context.Context, fromRepo, toRepo string, desc ocispec.Descriptor) error {
	repository, err := s.repository(ctx, toRepo)
	if err != nil {
		return err
	}

	named, err := ref.ParseNamed(fromRepo)
	if err != nil {
		return err
	}

	can, err := ref.WithDigest(named, desc.Digest)
	if err != nil {
		return err
	}

	blob, err := repository.Blobs(ctx).Create(ctx, registry.WithMountFrom(can))
	if blob != nil {
		return fmt.Errorf("Expected blob writer to be nil, was %v", blob)
	}

	// distribution will return the ErrBlobMounted error if the blob is already mounted.
	if ebm, ok := err.(distribution.ErrBlobMounted); ok {
		if ebm.From.Digest() != desc.Digest {
			return fmt.Errorf("Unexpected digest: %s, expected %s", ebm.From.Digest(), desc.Digest)
		}
		if ebm.From.Name() != fromRepo {
			return fmt.Errorf("Unexpected from: %s, expected %s", ebm.From.Name(), fromRepo)
		}
	} else {
		return fmt.Errorf("Unexpected error: %w, expected an ErrBlobMounted", err)
	}

	return nil
}

// StatBlob stats the blob in the storage.
func (s *storage) StatBlob(ctx context.Context, repo, digest string) (bool, error) {
	repository, err := s.repository(ctx, repo)
	if err != nil {
		return false, err
	}

	_, err = repository.Blobs(ctx).Stat(ctx, godigest.Digest(digest))
	if err != nil {
		// If the blob not found, distribution will return ErrBlobUnknown.
		if errors.Is(err, distribution.ErrBlobUnknown) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

// StatManifest stats the manifest in the storage.
func (s *storage) StatManifest(ctx context.Context, repo, digest string) (bool, error) {
	repository, err := s.repository(ctx, repo)
	if err != nil {
		return false, err
	}

	manifest, err := repository.Manifests(ctx)
	if err != nil {
		return false, err
	}

	return manifest.Exists(ctx, godigest.Digest(digest))
}

// ListRepositories lists all the repositories in the storage.
func (s *storage) ListRepositories(ctx context.Context) ([]string, error) {
	var repos []string
	if err := s.store.(distribution.RepositoryEnumerator).Enumerate(ctx, func(name string) error {
		repos = append(repos, name)
		return nil
	}); err != nil {
		return nil, err
	}

	return repos, nil
}

// ListTags lists all the tags in the repository.
func (s *storage) ListTags(ctx context.Context, repo string) ([]string, error) {
	repository, err := s.repository(ctx, repo)
	if err != nil {
		return nil, err
	}

	return repository.Tags(ctx).All(ctx)
}

// PerformGC performs the garbage collection in the storage to free up the space.
func (s *storage) PerformGC(ctx context.Context, dryRun, removeUntagged bool) error {
	return registry.MarkAndSweep(ctx, s.driver, s.store, registry.GCOpts{
		DryRun:         dryRun,
		RemoveUntagged: removeUntagged,
	})
}
