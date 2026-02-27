package mlflow

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/service/ml"
	log "github.com/sirupsen/logrus"
)

type MlFlowClient struct {
	registry *ml.ModelRegistryAPI
}

// TODO hdfs
// TODO sql
// TODO file
// Read https://mlflow.org/docs/latest/self-hosting/architecture/backend-store/
var (
	storageProvider = map[string]StorageBackend{
		"s3": &S3StorageBackend{},
	}
)

func NewMlFlowRegistry(mlflowClient *client.DatabricksClient) (MlFlowClient, error) {
	var registry *ml.ModelRegistryAPI

	if mlflowClient != nil {
		registry = ml.NewModelRegistry(mlflowClient)
		log.Info("using default mlflow client for mlflow registry API")
		return MlFlowClient{registry: registry}, nil
	}

	// TODO Support more auth methods?
	cfg := config.Config{
		// Credentials: config.BasicCredentials{},
	}
	mlClient, err := client.New(&cfg)
	if err != nil {
		return MlFlowClient{}, err
	}
	registry = ml.NewModelRegistry(mlClient)
	return MlFlowClient{registry: registry}, nil
}

// Pull latest modelVersion if `modelVersion` is nil
func (mlfr *MlFlowClient) PullModelByName(
	ctx context.Context,
	modelName string,
	modelVersion string,
	destSrc string,
) (string, error) {
	if mlfr == nil || mlfr.registry == nil {
		return "", errors.New("mlflow client is not initialized: registry is nil")
	}

	var pullVersion string

	if modelVersion == "" {
		versions, err := mlfr.registry.GetLatestVersionsAll(
			ctx,
			ml.GetLatestVersionsRequest{Name: modelName},
		)
		if err != nil {
			return "", errors.Join(fmt.Errorf("failed to get versions for model: %s", modelName), err)
		}

		if len(versions) == 0 {
			return "", fmt.Errorf("model %s has versions: %v", modelName, versions)
		}

		pullVersion = versions[0].Version

		log.WithFields(log.Fields{
			"model":   modelName,
			"version": pullVersion,
		}).Info("resolved model version")

	} else {

		all, err := mlfr.registry.SearchModelVersionsAll(ctx, ml.SearchModelVersionsRequest{})
		if err != nil {
			return "", errors.Join(errors.New("search model versions failed"), err)
		}
		var rawVersions []string

		for _, v := range all {
			rawVersions = append(rawVersions, v.Version)
		}

		if !slices.Contains(rawVersions, modelVersion) {
			msg := fmt.Sprintf(
				"model %s version %s not found, available version %v",
				modelName,
				modelVersion,
				rawVersions,
			)
			return "", errors.New(msg)
		} else {
			pullVersion = modelVersion
		}
	}
	log.WithField("version", pullVersion).Info("pulling model from model registry")

	uri, err := mlfr.registry.GetModelVersionDownloadUri(ctx, ml.GetModelVersionDownloadUriRequest{
		Name:    modelName,
		Version: pullVersion,
	})
	if err != nil {
		return "", errors.Join(errors.New("failed fetch download uri for model"), err)
	}
	log.WithField("artifact_uri", uri.ArtifactUri).Info("pulling model from artifact URI")
	parsed, err := url.Parse(uri.ArtifactUri)
	if err != nil {
		return "", fmt.Errorf("failed to parse artifact uri: %w", err)
	}
	if parsed == nil {
		return "", errors.New("failed to parse artifact uri")
	}

	switch parsed.Scheme {
	case "s3":
		s3storage := storageProvider[parsed.Scheme]
		destSrc = filepath.Join(destSrc, modelName)
		err = s3storage.DownloadModel(ctx, uri.ArtifactUri+"/", destSrc) // it's dir
		if err != nil {
			return "", err
		}
	default:
		msg := fmt.Sprintf("Unsupported artifact storage type: %s", parsed.Scheme)
		err = errors.New(msg)
		return "", err
	}

	log.Info("model downloaded")

	return destSrc, nil
}

type S3StorageBackend struct {
	addressing string
}

func (s3back *S3StorageBackend) DownloadModel(
	ctx context.Context,
	path string,
	destPath string,
) error {
	parsed, err := url.Parse(path)
	if err != nil {
		return err
	}

	bucketName := parsed.Host
	s3FolderPrefix := strings.TrimPrefix(parsed.Path, "/")
	log.WithFields(log.Fields{
		"bucket": bucketName,
		"path":   s3FolderPrefix,
	}).Info("parsed S3 artifact path")

	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		wrap := fmt.Errorf("Error loading AWS config, try change envs or profile: %v\n", err)
		return errors.Join(wrap, err)
	}

	log.WithFields(log.Fields{
		"region":   cfg.Region,
		"endpoint": aws.ToString(cfg.BaseEndpoint),
	}).Info("loaded AWS configuration")

	s3Client := s3.NewFromConfig(cfg)

	var partMiBs int64 = 10
	downloader := manager.NewDownloader(s3Client, func(d *manager.Downloader) {
		d.PartSize = partMiBs * 1024 * 1024
	})
	// List all objects under the prefix (including nested directories).
	paginator := s3.NewListObjectsV2Paginator(s3Client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
		Prefix: aws.String(s3FolderPrefix),
	})

	log.WithFields(log.Fields{
		"bucket": bucketName,
		"path":   s3FolderPrefix,
	}).Info("downloading artifacts from S3")

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			log.WithError(err).Error("failed to list S3 objects")
			return err
		}

		for _, object := range page.Contents {
			s3Key := *object.Key
			log.WithField("key", s3Key).Info("downloading S3 object")
			if strings.HasSuffix(s3Key, "/") { // Skip S3 "folder" markers
				continue
			}

			// Construct local file path
			relativePath := strings.TrimPrefix(s3Key, s3FolderPrefix)
			relativePath = strings.TrimPrefix(relativePath, "/")
			localFilePath := filepath.Join(destPath, relativePath)

			// Create local directories if they don't exist
			err = os.MkdirAll(filepath.Dir(localFilePath), 0o755)
			if err != nil {
				log.WithError(err).WithField("path", filepath.Dir(localFilePath)).
					Error("failed to create local directory")
				continue
			}

			// Download the object
			file, err := os.Create(localFilePath)
			if err != nil {
				log.WithError(err).WithField("path", localFilePath).Error("failed to create local file")
				continue
			}

			numBytes, err := downloader.Download(ctx, file, &s3.GetObjectInput{
				Bucket: aws.String(bucketName),
				Key:    aws.String(s3Key),
			})
			closeErr := file.Close()
			if err != nil || closeErr != nil {
				if err != nil {
					log.WithError(err).WithField("key", s3Key).Error("failed to download S3 object")
				}
				if closeErr != nil {
					log.WithError(closeErr).WithField("path", localFilePath).Error("failed to close local file")
				}
				if removeErr := os.Remove(localFilePath); removeErr != nil &&
					!errors.Is(removeErr, os.ErrNotExist) {
					log.WithError(removeErr).WithField("path", localFilePath).
						Error("failed to remove partial local file")
				}
				continue
			}
			log.WithFields(log.Fields{
				"key":   s3Key,
				"path":  localFilePath,
				"bytes": numBytes,
			}).Info("downloaded S3 object")
		}
	}

	return nil
}
