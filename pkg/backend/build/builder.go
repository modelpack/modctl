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

package build

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	modelspec "github.com/dragonflyoss/model-spec/specs-go/v1"
	sha256 "github.com/minio/sha256-simd"
	godigest "github.com/opencontainers/go-digest"
	spec "github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"

	"github.com/modelpack/modctl/internal/cache"
	buildconfig "github.com/modelpack/modctl/pkg/backend/build/config"
	"github.com/modelpack/modctl/pkg/backend/build/hooks"
	"github.com/modelpack/modctl/pkg/backend/build/interceptor"
	pkgcodec "github.com/modelpack/modctl/pkg/codec"
	"github.com/modelpack/modctl/pkg/storage"
)

// OutputType defines the type of output to generate.
type OutputType string

const (
	// OutputTypeLocal indicates that the output should be stored locally in modctl local storage.
	OutputTypeLocal OutputType = "local"
	// OutputTypeRemote indicates that the output should be pushed to a remote registry directly.
	OutputTypeRemote OutputType = "remote"
)

// Builder is an interface for building artifacts.
type Builder interface {
	// BuildLayer builds the layer blob from the given file path.
	BuildLayer(ctx context.Context, mediaType, workDir, path, destPath string, hooks hooks.Hooks) (ocispec.Descriptor, error)

	// BuildConfig builds the config blob of the artifact.
	BuildConfig(ctx context.Context, config modelspec.Model, hooks hooks.Hooks) (ocispec.Descriptor, error)

	// BuildManifest builds the manifest blob of the artifact.
	BuildManifest(ctx context.Context, layers []ocispec.Descriptor, config ocispec.Descriptor, annotations map[string]string, hooks hooks.Hooks) (ocispec.Descriptor, error)
}

type OutputStrategy interface {
	// OutputLayer outputs the layer blob to the storage (local or remote).
	OutputLayer(ctx context.Context, mediaType, relPath, destPath, digest string, size int64, reader io.Reader, hooks hooks.Hooks) (ocispec.Descriptor, error)

	// OutputConfig outputs the config blob to the storage (local or remote).
	OutputConfig(ctx context.Context, mediaType, digest string, size int64, reader io.Reader, hooks hooks.Hooks) (ocispec.Descriptor, error)

	// OutputManifest outputs the manifest blob to the storage (local or remote).
	OutputManifest(ctx context.Context, mediaType, digest string, size int64, reader io.Reader, hooks hooks.Hooks) (ocispec.Descriptor, error)
}

// NewBuilder creates a new builder instance.
func NewBuilder(outputType OutputType, store storage.Storage, repo, tag string, opts ...Option) (Builder, error) {
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}

	var (
		strategy OutputStrategy
		err      error
	)
	switch outputType {
	case OutputTypeLocal:
		strategy, err = NewLocalOutput(cfg, store, repo, tag)
	case OutputTypeRemote:
		strategy, err = NewRemoteOutput(cfg, repo, tag)
	default:
		return nil, fmt.Errorf("unsupported output type: %s", outputType)
	}

	if err != nil {
		return nil, err
	}

	// TODO: Use the storage dir specified from user.
	cache, err := cache.New(os.TempDir())
	if err != nil {
		// Just print the error message because cache is not critical.
		logrus.Errorf("failed to create cache: %v", err)
	}

	return &abstractBuilder{
		store:       store,
		repo:        repo,
		tag:         tag,
		strategy:    strategy,
		interceptor: cfg.interceptor,
		cache:       cache,
	}, nil
}

// abstractBuilder is an abstract implementation of the Builder interface.
type abstractBuilder struct {
	store storage.Storage
	repo  string
	tag   string
	// strategy is the output strategy used to output the blob.
	strategy OutputStrategy
	// interceptor is the interceptor used to intercept the build process.
	interceptor interceptor.Interceptor
	// cache is the cache used to store the file digest.
	cache cache.Cache
}

func (ab *abstractBuilder) BuildLayer(ctx context.Context, mediaType, workDir, path, destPath string, hooks hooks.Hooks) (ocispec.Descriptor, error) {
	info, err := os.Stat(path)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to get file info: %w", err)
	}

	if info.IsDir() {
		return ocispec.Descriptor{}, fmt.Errorf("%s is a directory and not supported yet", path)
	}

	workDirPath, err := filepath.Abs(workDir)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to get absolute path of workDir: %w", err)
	}

	// Gets the relative path of the file as annotation.
	//nolint:typecheck
	relPath, err := filepath.Rel(workDirPath, path)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to get relative path: %w", err)
	}

	codec, err := pkgcodec.New(pkgcodec.TypeFromMediaType(mediaType))
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to create codec: %w", err)
	}

	logrus.Debugf("builder: starting build layer for file %s", relPath)

	// Encode the content by codec depends on the media type.
	reader, err := codec.Encode(path, workDirPath)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to encode file: %w", err)
	}

	reader, digest, size, err := ab.computeDigestAndSize(ctx, mediaType, path, workDirPath, info, reader, codec)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to compute digest and size: %w", err)
	}

	var (
		wg        sync.WaitGroup
		itErr     error
		applyDesc interceptor.ApplyDescriptorFn
	)
	// Intercept the reader if needed.
	if ab.interceptor != nil {
		var itReader io.Reader
		reader, itReader = splitReader(reader)

		wg.Add(1)
		go func() {
			defer wg.Done()
			applyDesc, itErr = ab.interceptor.Intercept(ctx, mediaType, relPath, codec.Type(), itReader)
		}()
	}

	desc, err := ab.strategy.OutputLayer(ctx, mediaType, relPath, destPath, digest, size, reader, hooks)
	if err != nil {
		return desc, err
	}

	// Wait for the interceptor to finish.
	wg.Wait()
	if itErr != nil {
		return desc, itErr
	}

	if applyDesc != nil {
		applyDesc(&desc)
	}

	// Add file metadata to descriptor.
	if err := addFileMetadata(&desc, path, relPath); err != nil {
		return desc, err
	}

	return desc, nil
}

func (ab *abstractBuilder) BuildConfig(ctx context.Context, config modelspec.Model, hooks hooks.Hooks) (ocispec.Descriptor, error) {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to marshal config: %w", err)
	}

	digest := fmt.Sprintf("sha256:%x", sha256.Sum256(configJSON))
	return ab.strategy.OutputConfig(ctx, modelspec.MediaTypeModelConfig, digest, int64(len(configJSON)), bytes.NewReader(configJSON), hooks)
}

func (ab *abstractBuilder) BuildManifest(ctx context.Context, layers []ocispec.Descriptor, config ocispec.Descriptor, annotations map[string]string, hooks hooks.Hooks) (ocispec.Descriptor, error) {
	manifest := &ocispec.Manifest{
		Versioned: spec.Versioned{
			SchemaVersion: 2,
		},
		Annotations:  annotations,
		ArtifactType: modelspec.ArtifactTypeModelManifest,
		Config: ocispec.Descriptor{
			MediaType: config.MediaType,
			Digest:    config.Digest,
			Size:      config.Size,
		},
		MediaType: ocispec.MediaTypeImageManifest,
		Layers:    layers,
	}

	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to marshal manifest: %w", err)
	}

	digest := fmt.Sprintf("sha256:%x", sha256.Sum256(manifestJSON))
	return ab.strategy.OutputManifest(ctx, manifest.MediaType, digest, int64(len(manifestJSON)), bytes.NewReader(manifestJSON), hooks)
}

// computeDigestAndSize computes the digest and size for the encoded content, using cache if available.
func (ab *abstractBuilder) computeDigestAndSize(ctx context.Context, mediaType, path, workDirPath string, info os.FileInfo, reader io.Reader, codec pkgcodec.Codec) (io.Reader, string, int64, error) {
	// Try to retrieve valid digest from cache for raw model weights.
	if mediaType == modelspec.MediaTypeModelWeightRaw {
		if digest, size, ok := ab.retrieveCache(ctx, path, info); ok {
			return reader, digest, size, nil
		}
	}

	logrus.Infof("builder: calculating digest for file %s", path)

	hash := sha256.New()
	size, err := io.Copy(hash, reader)
	if err != nil {
		return reader, "", 0, fmt.Errorf("failed to copy content to hash: %w", err)
	}
	digest := fmt.Sprintf("sha256:%x", hash.Sum(nil))

	logrus.Infof("builder: calculated digest for file %s [digest: %s]", path, digest)

	// Reset reader for subsequent use.
	reader, err = resetReader(reader, path, workDirPath, codec)
	if err != nil {
		return reader, "", 0, err
	}

	// Update cache.
	if mediaType == modelspec.MediaTypeModelWeightRaw {
		if err := ab.updateCache(ctx, path, info.ModTime(), size, digest); err != nil {
			logrus.Warnf("builder: failed to update cache for file %s: %s", path, err)
		}
	}

	return reader, digest, size, nil
}

// retrieveCache checks if mtime and size match, then returns the cached digest.
func (ab *abstractBuilder) retrieveCache(ctx context.Context, path string, info os.FileInfo) (string, int64, bool) {
	if ab.cache == nil {
		return "", 0, false
	}

	item, err := ab.cache.Get(ctx, path)
	if err != nil {
		if !errors.Is(err, cache.ErrNotFound) {
			logrus.Errorf("builder: failed to retrieve cache item for file %s: %s", path, err)
		}

		return "", 0, false
	}

	if item.ModTime != info.ModTime() || item.Size != info.Size() {
		logrus.Warnf("builder: cache item for file %s is stale, skip cache", path)
		return "", 0, false
	}

	logrus.Infof("builder: retrieved from cache for file %s [digest: %s]", path, item.Digest)
	return item.Digest, item.Size, true
}

// updateCache writes mtime, size, and digest to cache.
func (ab *abstractBuilder) updateCache(ctx context.Context, path string, mtime time.Time, size int64, digest string) error {
	if ab.cache == nil {
		return errors.New("cache is not initialized")
	}

	item := &cache.Item{
		Path:      path,
		ModTime:   mtime,
		Size:      size,
		Digest:    digest,
		CreatedAt: time.Now(),
	}

	return ab.cache.Put(ctx, item)
}

// BuildModelConfig builds the model config.
func BuildModelConfig(modelConfig *buildconfig.Model, layers []ocispec.Descriptor) (modelspec.Model, error) {
	if modelConfig == nil {
		return modelspec.Model{}, fmt.Errorf("model config is nil")
	}

	config := modelspec.ModelConfig{
		Architecture: modelConfig.Architecture,
		Format:       modelConfig.Format,
		Precision:    modelConfig.Precision,
		Quantization: modelConfig.Quantization,
		ParamSize:    modelConfig.ParamSize,
	}

	if modelConfig.Reasoning {
		config.Capabilities = &modelspec.ModelCapabilities{
			Reasoning: &modelConfig.Reasoning,
		}
	}

	descriptor := modelspec.ModelDescriptor{
		Family:    modelConfig.Family,
		Name:      modelConfig.Name,
		SourceURL: modelConfig.SourceURL,
		Revision:  modelConfig.SourceRevision,
	}

	if !modelConfig.NoCreationTime {
		createdAt := time.Now()
		descriptor.CreatedAt = &createdAt
	}

	diffIDs := make([]godigest.Digest, 0, len(layers))
	for _, layer := range layers {
		diffIDs = append(diffIDs, layer.Digest)
	}

	fs := modelspec.ModelFS{
		Type:    "layers",
		DiffIDs: diffIDs,
	}

	return modelspec.Model{
		Config:     config,
		Descriptor: descriptor,
		ModelFS:    fs,
	}, nil
}

// resetReader resets the reader to the beginning or re-encodes if not seekable.
func resetReader(reader io.Reader, path, workDirPath string, codec pkgcodec.Codec) (io.Reader, error) {
	if seeker, ok := reader.(io.ReadSeeker); ok {
		logrus.Debugf("builder: seeking reader to beginning for file %s", path)
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			return nil, fmt.Errorf("failed to seek reader: %w", err)
		}
		return reader, nil
	}

	logrus.Debugf("builder: reader not seekable, re-encoding file %s", path)
	return codec.Encode(path, workDirPath)
}

// addFileMetadata adds file metadata to the descriptor.
func addFileMetadata(desc *ocispec.Descriptor, path, relPath string) error {
	metadata, err := getFileMetadata(path)
	if err != nil {
		return fmt.Errorf("failed to retrieve file metadata: %w", err)
	}

	metadataStr, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	logrus.Infof("builder: retrieved metadata for file %s [metadata: %s]", relPath, string(metadataStr))

	if desc.Annotations == nil {
		desc.Annotations = make(map[string]string)
	}
	desc.Annotations[modelspec.AnnotationFileMetadata] = string(metadataStr)
	return nil
}

// splitReader splits the original reader into two readers.
func splitReader(original io.Reader) (io.Reader, io.Reader) {
	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()
	multiWriter := io.MultiWriter(w1, w2)

	go func() {
		defer w1.Close()
		defer w2.Close()

		_, err := io.Copy(multiWriter, original)
		if err != nil {
			w1.CloseWithError(err)
			w2.CloseWithError(err)
		}
	}()

	return r1, r2
}

// getFileMetadata retrieves metadata for a file at the given path.
func getFileMetadata(path string) (modelspec.FileMetadata, error) {
	var metadata modelspec.FileMetadata

	info, err := os.Stat(path)
	if err != nil {
		return metadata, err
	}

	metadata.Name = info.Name()
	metadata.Mode = uint32(info.Mode().Perm())
	metadata.Size = info.Size()
	metadata.ModTime = info.ModTime()
	// Set Typeflag.
	switch {
	case info.Mode().IsRegular():
		metadata.Typeflag = 0 // Regular file
	case info.Mode().IsDir():
		metadata.Typeflag = 5 // Directory
	case info.Mode()&os.ModeSymlink != 0:
		metadata.Typeflag = 2 // Symlink
	default:
		return metadata, errors.New("unknown file typeflag")
	}

	// UID and GID (Unix-specific).
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		metadata.Uid = stat.Uid
		metadata.Gid = stat.Gid
	}

	return metadata, nil
}
