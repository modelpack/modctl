package mlflow

import (
	"context"
	"errors"
	"fmt"
	"log"
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
		fmt.Println("Use default mlflow client for MlFlowRegistryAPI")
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

func (mlfr *MlFlowClient) PullModelByName(
	ctx context.Context,
	modelName string,
	modelVersion string,
	destSrc string,
) (string, error) {
	if mlfr == nil || mlfr.registry == nil {
		return "", errors.New("mlflow client is not initialized: registry is nil")
	}

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

	var rawVersion []string
	for _, version := range versions {
		rawVersion = append(rawVersion, version.Version)
	}
	contains := slices.Contains(rawVersion, modelVersion)
	if !contains {
		msg := fmt.Sprintf(
			"model %s version %s not found, available version %v",
			modelName,
			modelVersion,
			rawVersion,
		)
		return "", errors.New(msg)
	}

	fmt.Printf("Found versions: '%v' for model '%s'\n", rawVersion, modelName)
	if modelVersion == "" {
		modelVersion = rawVersion[0]
	}

	uri, err := mlfr.registry.GetModelVersionDownloadUri(ctx, ml.GetModelVersionDownloadUriRequest{
		Name:    modelName,
		Version: modelVersion,
	})
	if err != nil {
		return "", err
	}
	fmt.Printf("Try pull model from uri %s", uri.ArtifactUri)
	parsed, err := url.Parse(uri.ArtifactUri)
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

	fmt.Printf("✅ Model downloaded")

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
	s3FolderPrefix := parsed.Path[1:]
	fmt.Printf("Parsed s3 bucket %s, path %s from path", bucketName, s3FolderPrefix)

	cfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		wrap := fmt.Errorf("Error loading AWS config, try change envs or profile: %v\n", err)
		return errors.Join(wrap, err)
	}

	log.Printf("Region - %s, endpoint - %s", cfg.Region, aws.ToString(cfg.BaseEndpoint))

	s3Client := s3.NewFromConfig(cfg)

	var partMiBs int64 = 10
	downloader := manager.NewDownloader(s3Client, func(d *manager.Downloader) {
		d.PartSize = partMiBs * 1024 * 1024
	})
	// List objects with the specified prefix
	paginator := s3.NewListObjectsV2Paginator(s3Client, &s3.ListObjectsV2Input{
		Bucket:    aws.String(bucketName),
		Prefix:    aws.String(s3FolderPrefix),
		Delimiter: aws.String("/"),
	})

	log.Printf("Start downloading from s3 bucket %s, path %s", bucketName, s3FolderPrefix)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			log.Printf("Error listing objects: %v\n", err)
			return err
		}

		for _, object := range page.Contents {
			s3Key := *object.Key
			log.Printf("Downloading object: %s\n", s3Key)
			//if strings.HasSuffix(s3Key, "/") { // Skip S3 "folder" markers
			//	continue
			//}

			// Construct local file path
			relativePath := strings.TrimPrefix(s3Key, s3FolderPrefix)
			localFilePath := filepath.Join(destPath, relativePath)

			// Create local directories if they don't exist
			err = os.MkdirAll(filepath.Dir(localFilePath), 0o755)
			if err != nil {
				log.Printf(
					"Error creating local directory %s: %v\n",
					filepath.Dir(localFilePath),
					err,
				)
				continue
			}

			// Download the object
			file, err := os.Create(localFilePath)
			if err != nil {
				log.Printf("Error creating local file %s: %v\n", localFilePath, err)
				continue
			}
			defer file.Close()

			numBytes, err := downloader.Download(ctx, file, &s3.GetObjectInput{
				Bucket: aws.String(bucketName),
				Key:    aws.String(s3Key),
			})
			if err != nil {
				log.Printf("Error downloading object %s: %v\n", s3Key, err)
				continue
			}
			log.Printf("Downloaded %s to %s (%d bytes)\n", s3Key, localFilePath, numBytes)
		}
	}

	return nil
}
