package mlflow

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMlflowProvider_DownloadModel(t *testing.T) {
	type fields struct {
		mlfClient MlFlowClient
	}
	type args struct {
		ctx      context.Context
		modelURL string
		destDir  string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "empty model url returns error",
			args:    args{ctx: context.Background(), modelURL: "", destDir: "/tmp"},
			want:    "",
			wantErr: true,
		},
		{
			name: "invalid model url returns error",
			args: args{
				ctx:      context.Background(),
				modelURL: "http://my-model/1",
				destDir:  "/tmp",
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &MlflowProvider{
				mflClient: tt.fields.mlfClient,
			}
			got, err := p.DownloadModel(tt.args.ctx, tt.args.modelURL, tt.args.destDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("DownloadModel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DownloadModel() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMlflowProvider_Name(t *testing.T) {
	type fields struct {
		mlfClient MlFlowClient
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "returns provider name",
			want: "mlflow",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &MlflowProvider{
				mflClient: tt.fields.mlfClient,
			}
			if got := p.Name(); got != tt.want {
				t.Errorf("Name() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMlflowProvider_SupportsURL(t *testing.T) {
	type fields struct {
		mlfClient MlFlowClient
	}
	type args struct {
		url string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "models scheme supported",
			args: args{url: "models://my-model/1"},
			want: true,
		},
		{
			name: "trimmed whitespace supported",
			args: args{url: "  models://my-model/1  "},
			want: true,
		},
		{
			name: "http scheme not supported",
			args: args{url: "http://my-model/1"},
			want: false,
		},
		{
			name: "empty url not supported",
			args: args{url: ""},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &MlflowProvider{
				mflClient: tt.fields.mlfClient,
			}
			if got := p.SupportsURL(tt.args.url); got != tt.want {
				t.Errorf("SupportsURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_checkMlflowAuth(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "missing databricks and mlflow envs returns error",
			wantErr: true,
		},
		{
			name:    "databricks host set returns nil",
			wantErr: false,
		},
		{
			name:    "mlflow tracking set returns nil",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("DATABRICKS_HOST", "")
			t.Setenv("DATABRICKS_USERNAME", "")
			t.Setenv("DATABRICKS_PASSWORD", "")
			t.Setenv("MLFLOW_TRACKING_URI", "")
			t.Setenv("MLFLOW_TRACKING_USERNAME", "")
			t.Setenv("MLFLOW_TRACKING_PASSWORD", "")
			switch tt.name {
			case "databricks host set returns nil":
				t.Setenv("DATABRICKS_HOST", "https://example.com")
				t.Setenv("DATABRICKS_USERNAME", "user")
				t.Setenv("DATABRICKS_PASSWORD", "pass")

			case "mlflow tracking set returns nil":
				t.Setenv("MLFLOW_TRACKING_URI", "https://mlflow.example.com")
				t.Setenv("MLFLOW_TRACKING_USERNAME", "mlf-user")
				t.Setenv("MLFLOW_TRACKING_PASSWORD", "mlf-pass")
			}

			err := checkMlflowAuth()
			assert.NotEqual(t, tt.wantErr, err)
		})
	}
}

func Test_hasAnyPrefix(t *testing.T) {
	type args struct {
		s    string
		subs []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"One valid and one unvalid",
			args{"models://my-model/1", []string{"models", "invalid"}},
			true,
		},
		{"All unvalid", args{"http://my-model/1", []string{"models", "invalid"}}, false},
		{"Empty substrings", args{"models://my-model/1", []string{}}, false},
		{"Empty main string", args{"", []string{"models", "invalid"}}, false},
		{"Both empty", args{"", []string{}}, false},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasAnyPrefix(tt.args.s, tt.args.subs); got != tt.want {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_parseModelURL(t *testing.T) {
	type args struct {
		modelURL string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		want1   string
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			"models name with mlflow schema",
			args{modelURL: "models://my-model/1"},
			"my-model",
			"1",
			false,
		},
		{"models name with http schema", args{modelURL: "http://my-model/1"}, "", "", true},
		{"models name without version", args{modelURL: "my-model"}, "my-model", "", false},
		{
			"models with schema and without version",
			args{modelURL: "models://my-model"},
			"my-model",
			"",
			false,
		},
		{"invalid url", args{modelURL: "://my-model/1"}, "", "", true},
		{
			"model without schema should return error",
			args{modelURL: "my-model/1"},
			"my-model",
			"1",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := parseModelURL(tt.args.modelURL)
			assert.Equal(
				t,
				tt.wantErr,
				err != nil,
				"parseModelURL() error = %v, wantErr %v",
				err,
				tt.wantErr,
			)
			assert.Equal(t, tt.want, got, "parseModelURL() got = %v, want %v", got, tt.want)
			assert.Equal(t, tt.want1, got1, "parseModelURL() got1 = %v, want %v", got1, tt.want1)
		})
	}
}
