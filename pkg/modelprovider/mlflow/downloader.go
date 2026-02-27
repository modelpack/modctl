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
	"github.com/sirupsen/logrus"
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
		logrus.Infof("mlflow: using default client for MlFlowRegistryAPI")
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

		logrus.Infof("mlflow: found versions '%v' for model '%s'", pullVersion, modelName)

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
	logrus.Infof("mlflow: starting pull model from registry with version %s", pullVersion)

	uri, err := mlfr.registry.GetModelVersionDownloadUri(ctx, ml.GetModelVersionDownloadUriRequest{
		Name:    modelName,
		Version: pullVersion,
	})
	if err != nil {
		return "", errors.Join(errors.New("failed fetch download uri for model"), err)
	}
	logrus.Infof("mlflow: pulling model from uri %s", uri.ArtifactUri)
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

	logrus.Infof("mlflow: model downloaded successfully")

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
	logrus.Debugf("mlflow: parsed s3 bucket %s, path %s", bucketName, s3FolderPrefix)

	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		wrap := fmt.Errorf("Error loading AWS config, try change envs or profile: %v\n", err)
		return errors.Join(wrap, err)
	}

	logrus.Debugf("mlflow: aws region %s, endpoint %s", cfg.Region, aws.ToString(cfg.BaseEndpoint))

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

	logrus.Infof("mlflow: starting download from s3 bucket %s, path %s", bucketName, s3FolderPrefix)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			logrus.Errorf("mlflow: failed to list objects: %v", err)
			return err
		}

		for _, object := range page.Contents {
			s3Key := *object.Key
			logrus.Debugf("mlflow: downloading object %s", s3Key)
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
				logrus.Errorf(
					"mlflow: failed to create local directory %s: %v",
					filepath.Dir(localFilePath), err,
				)
				continue
			}

			// Download the object
			file, err := os.Create(localFilePath)
			if err != nil {
				logrus.Errorf("mlflow: failed to create local file %s: %v", localFilePath, err)
				continue
			}

			numBytes, err := downloader.Download(ctx, file, &s3.GetObjectInput{
				Bucket: aws.String(bucketName),
				Key:    aws.String(s3Key),
			})
			closeErr := file.Close()
			if err != nil || closeErr != nil {
				if err != nil {
					logrus.Errorf("mlflow: failed to download object %s: %v", s3Key, err)
				}
				if closeErr != nil {
					logrus.Errorf("mlflow: failed to close file %s: %v", localFilePath, closeErr)
				}
				if removeErr := os.Remove(localFilePath); removeErr != nil &&
					!errors.Is(removeErr, os.ErrNotExist) {
					logrus.Errorf(
						"mlflow: failed to remove partial file %s: %v",
						localFilePath, removeErr,
					)
				}
				continue
			}
			logrus.Debugf("mlflow: downloaded %s to %s (%d bytes)", s3Key, localFilePath, numBytes)
		}
	}

	return nil
}
