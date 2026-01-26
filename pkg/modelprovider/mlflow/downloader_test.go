package mlflow

import (
	"context"
	"testing"

	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

func TestMlFlowClient_PullModelByName(t *testing.T) {
	type fields struct {
		registry *ml.ModelRegistryAPI
	}
	type args struct {
		ctx          context.Context
		modelName    string
		modelVersion string
		destSrc      string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "nil receiver returns error",
			fields:  fields{registry: nil},
			args:    args{ctx: context.Background(), modelName: "model", modelVersion: "1", destSrc: "/tmp"},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mlfr := &MlFlowClient{
				registry: tt.fields.registry,
			}
			got, err := mlfr.PullModelByName(tt.args.ctx, tt.args.modelName, tt.args.modelVersion, tt.args.destSrc)
			if (err != nil) != tt.wantErr {
				t.Errorf("PullModelByName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("PullModelByName() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewMlFlowRegistry(t *testing.T) {
	type args struct {
		mlflowClient *client.DatabricksClient
	}
	tests := []struct {
		name    string
		args    args
		want    MlFlowClient
		wantErr bool
	}{
		{
			name:    "non-nil client returns registry",
			args:    args{mlflowClient: &client.DatabricksClient{}},
			want:    MlFlowClient{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewMlFlowRegistry(tt.args.mlflowClient)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewMlFlowRegistry() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.registry == nil {
				t.Errorf("NewMlFlowRegistry() registry is nil")
			}
		})
	}
}

func TestS3StorageBackend_DownloadModel(t *testing.T) {
	type fields struct {
		addressing string
	}
	type args struct {
		ctx      context.Context
		path     string
		destPath string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "invalid url returns error",
			fields:  fields{addressing: ""},
			args:    args{ctx: context.Background(), path: "http://[::1", destPath: "/tmp"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s3back := &S3StorageBackend{
				addressing: tt.fields.addressing,
			}
			if err := s3back.DownloadModel(tt.args.ctx, tt.args.path, tt.args.destPath); (err != nil) != tt.wantErr {
				t.Errorf("DownloadModel() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
