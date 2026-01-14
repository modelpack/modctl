package mlflow

import "context"

type StorageBackend interface {
	DownloadModel(ctx context.Context, path string, destPath string) error
}
