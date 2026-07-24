package storage

import (
	"testing"

	"github.com/HuolalaTech/page-spy-api/config"
)

func TestRemoteAPIInitializesReusableClient(t *testing.T) {
	api, err := newRemoteApi(&config.StorageConfig{
		Region:   "us-east-1",
		Endpoint: "http://127.0.0.1:9000",
		Bucket:   "test",
	})
	if err != nil {
		t.Fatalf("create remote API: %v", err)
	}
	if api.client == nil {
		t.Fatal("S3 client was not initialized")
	}
}
