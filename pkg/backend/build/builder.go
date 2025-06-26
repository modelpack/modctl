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
	"strconv"
	"sync"
	"syscall"
	"time"

	modelspec "github.com/CloudNativeAI/model-spec/specs-go/v1"
	sha256 "github.com/minio/sha256-simd"
	godigest "github.com/opencontainers/go-digest"
	spec "github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"

	buildconfig "github.com/CloudNativeAI/modctl/pkg/backend/build/config"
	"github.com/CloudNativeAI/modctl/pkg/backend/build/hooks"
	"github.com/CloudNativeAI/modctl/pkg/backend/build/interceptor"
	pkgcodec "github.com/CloudNativeAI/modctl/pkg/codec"
	"github.com/CloudNativeAI/modctl/pkg/storage"
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
	BuildLayer(ctx context.Context, mediaType, workDir, path string, hooks hooks.Hooks) (ocispec.Descriptor, error)

	// BuildConfig builds the config blob of the artifact.
	BuildConfig(ctx context.Context, layers []ocispec.Descriptor, modelConfig *buildconfig.Model, hooks hooks.Hooks) (ocispec.Descriptor, error)

	// BuildManifest builds the manifest blob of the artifact.
	BuildManifest(ctx context.Context, layers []ocispec.Descriptor, config ocispec.Descriptor, annotations map[string]string, hooks hooks.Hooks) (ocispec.Descriptor, error)
}

type OutputStrategy interface {
	// OutputLayer outputs the layer blob to the storage (local or remote).
	OutputLayer(ctx context.Context, mediaType, relPath, digest string, size int64, reader io.Reader, hooks hooks.Hooks) (ocispec.Descriptor, error)

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

	return &abstractBuilder{
		store:       store,
		repo:        repo,
		tag:         tag,
		strategy:    strategy,
		interceptor: cfg.interceptor,
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
}

func (ab *abstractBuilder) BuildLayer(ctx context.Context, mediaType, workDir, path string, hooks hooks.Hooks) (ocispec.Descriptor, error) {
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

	reader, digest, size, err := computeDigestAndSize(mediaType, path, workDirPath, info, reader, codec)
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

	desc, err := ab.strategy.OutputLayer(ctx, mediaType, relPath, digest, size, reader, hooks)
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

func (ab *abstractBuilder) BuildConfig(ctx context.Context, layers []ocispec.Descriptor, modelConfig *buildconfig.Model, hooks hooks.Hooks) (ocispec.Descriptor, error) {
	config, err := buildModelConfig(modelConfig, layers)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to build model config: %w", err)
	}

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

// buildModelConfig builds the model config.
func buildModelConfig(modelConfig *buildconfig.Model, layers []ocispec.Descriptor) (*modelspec.Model, error) {
	if modelConfig == nil {
		return nil, fmt.Errorf("model config is nil")
	}

	config := modelspec.ModelConfig{
		Architecture: modelConfig.Architecture,
		Format:       modelConfig.Format,
		Precision:    modelConfig.Precision,
		Quantization: modelConfig.Quantization,
		ParamSize:    modelConfig.ParamSize,
	}

	createdAt := time.Now()
	descriptor := modelspec.ModelDescriptor{
		CreatedAt: &createdAt,
		Family:    modelConfig.Family,
		Name:      modelConfig.Name,
		SourceURL: modelConfig.SourceURL,
		Revision:  modelConfig.SourceRevision,
	}

	diffIDs := make([]godigest.Digest, 0, len(layers))
	for _, layer := range layers {
		diffIDs = append(diffIDs, layer.Digest)
	}

	fs := modelspec.ModelFS{
		Type:    "layers",
		DiffIDs: diffIDs,
	}

	return &modelspec.Model{
		Config:     config,
		Descriptor: descriptor,
		ModelFS:    fs,
	}, nil
}

// computeDigestAndSize computes the digest and size for the encoded content, using xattrs if available.
func computeDigestAndSize(mediaType, path, workDirPath string, info os.FileInfo, reader io.Reader, codec pkgcodec.Codec) (io.Reader, string, int64, error) {
	var digest string
	var size int64

	if pkgcodec.IsRawMediaType(mediaType) {
		// By default let's assume the mtime and size has changed.
		mtimeChanged := true
		sizeChanged := true

		if mtime, err := getXattr(path, xattrMtimeKey(mediaType)); err == nil {
			if string(mtime) == fmt.Sprintf("%d", info.ModTime().UnixNano()) {
				mtimeChanged = false
			}
		}

		if sizeBytes, err := getXattr(path, xattrSizeKey(mediaType)); err == nil {
			if parsedSize, err := strconv.ParseInt(string(sizeBytes), 10, 64); err == nil {
				if parsedSize == info.Size() {
					sizeChanged = false
				}
			}
		}

		if !mtimeChanged && !sizeChanged {
			// Check xattrs for cached digest and size.
			if sha256, err := getXattr(path, xattrSha256Key(mediaType)); err == nil {
				digest = string(sha256)
				logrus.Infof("builder: retrieved sha256 hash from xattr for file %s [digest: %s]", path, digest)
			}

			if sizeBytes, err := getXattr(path, xattrSizeKey(mediaType)); err == nil {
				if parsedSize, err := strconv.ParseInt(string(sizeBytes), 10, 64); err == nil {
					size = parsedSize
					logrus.Infof("builder: retrieved size from xattr for file %s [size: %d]", path, size)
				}
			}
		}
	}

	// Compute digest and size if not retrieved from xattrs.
	if digest == "" {
		logrus.Infof("builder: calculating digest for file %s", path)
		var err error
		hash := sha256.New()
		size, err = io.Copy(hash, reader)
		if err != nil {
			return reader, "", 0, fmt.Errorf("failed to copy content to hash: %w", err)
		}
		digest = fmt.Sprintf("sha256:%x", hash.Sum(nil))
		logrus.Infof("builder: calculated digest for file %s [digest: %s]", path, digest)

		// Reset reader
		reader, err = resetReader(reader, path, workDirPath, codec)
		if err != nil {
			return reader, "", 0, err
		}

		// Store xattrs if raw media type.
		if pkgcodec.IsRawMediaType(mediaType) {
			setXattr(path, xattrMtimeKey(mediaType), fmt.Appendf([]byte{}, "%d", info.ModTime().UnixNano()))
			setXattr(path, xattrSha256Key(mediaType), []byte(digest))
			setXattr(path, xattrSizeKey(mediaType), fmt.Appendf([]byte{}, "%d", size))
		}
	}

	return reader, digest, size, nil
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

func xattrSha256Key(mediaType string) string {
	// Uniformity between linux and mac platforms is simplified by adding the prefix 'user.',
	// because the key may be unlimited under mac,
	// but on linux, in some cases, the user can only manipulate the user space.
	return fmt.Sprintf("user.%s.sha256", mediaType)
}

func xattrSizeKey(mediaType string) string {
	// Uniformity between linux and mac platforms is simplified by adding the prefix 'user.',
	// because the key may be unlimited under mac,
	// but on linux, in some cases, the user can only manipulate the user space.
	return fmt.Sprintf("user.%s.size", mediaType)
}

func xattrMtimeKey(mediaType string) string {
	// Uniformity between linux and mac platforms is simplified by adding the prefix 'user.',
	// because the key may be unlimited under mac,
	// but on linux, in some cases, the user can only manipulate the user space.
	return fmt.Sprintf("user.%s.mtime", mediaType)
}

// getXattr retrieves an xattr value for a given key.
func getXattr(path, key string) ([]byte, error) {
	var value []byte
	sz, err := unix.Getxattr(path, key, value)
	if err != nil {
		logrus.Warnf("builder: failed to get xattr %s for file %s: %v", key, path, err)
		return nil, err
	}

	value = make([]byte, sz)
	_, err = unix.Getxattr(path, key, value)
	if err != nil {
		logrus.Warnf("builder: failed to get xattr %s for file %s: %v", key, path, err)
		return nil, err
	}

	return value, nil
}

// setXattr sets an xattr value for a given key.
func setXattr(path, key string, value []byte) {
	if err := unix.Setxattr(path, key, value, 0); err != nil {
		logrus.Warnf("builder: failed to set xattr %s for file %s: %v", key, path, err)
	} else {
		logrus.Infof("builder: set xattr %s for file %s: %s", key, path, string(value))
	}
}
