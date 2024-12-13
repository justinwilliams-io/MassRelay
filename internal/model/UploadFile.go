package model

import "context"

type Config struct {
	RemoteURL            string `yaml:"remote_url"`
	MaxConcurrentUploads int    `yaml:"max_concurrent_uploads"`
	LogLevel             string `yaml:"log_level"`
	Token                string `yaml:"token"`
}

type Uploader interface {
    UploadFile(ctx context.Context, file string, url string, authToken string, queryParams map[string]string) error
}
