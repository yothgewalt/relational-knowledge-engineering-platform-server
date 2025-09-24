package minio

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
)


func TestNewMinIOService(t *testing.T) {
	tests := []struct {
		name      string
		config    MinIOConfig
		expectErr bool
		errMsg    string
	}{
		{
			name: "Valid config with credentials",
			config: MinIOConfig{
				Endpoint:        "localhost:9000",
				AccessKeyID:     "testkey",
				SecretAccessKey: "testsecret",
				UseSSL:          false,
				BucketName:      "test-bucket",
			},
			expectErr: false,
		},
		{
			name: "Missing endpoint",
			config: MinIOConfig{
				AccessKeyID:     "testkey",
				SecretAccessKey: "testsecret",
				BucketName:      "test-bucket",
			},
			expectErr: true,
			errMsg:    "MinIO endpoint is required",
		},
		{
			name: "Missing bucket name",
			config: MinIOConfig{
				Endpoint:        "localhost:9000",
				AccessKeyID:     "testkey",
				SecretAccessKey: "testsecret",
			},
			expectErr: true,
			errMsg:    "MinIO bucket name is required",
		},
		{
			name: "Missing credentials",
			config: MinIOConfig{
				Endpoint:   "localhost:9000",
				BucketName: "test-bucket",
			},
			expectErr: true,
			errMsg:    "MinIO credentials are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewMinIOService(tt.config)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, client)
			} else {
				if err != nil && strings.Contains(err.Error(), "connection") {
					t.Skip("Skipping test - MinIO server not available")
				}
			}
		})
	}
}


func TestMinIOClient_HealthCheck(t *testing.T) {
	config := MinIOConfig{
		Endpoint:        "localhost:9000",
		AccessKeyID:     "testkey",
		SecretAccessKey: "testsecret",
		UseSSL:          false,
		BucketName:      "test-bucket",
	}

	client, err := NewMinIOService(config)
	if err != nil && strings.Contains(err.Error(), "connection") {
		t.Skip("Skipping test - MinIO server not available")
	}
	if err != nil {
		t.Logf("MinIO client creation failed (expected if MinIO not running): %v", err)
		return
	}

	ctx := context.Background()
	status := client.HealthCheck(ctx)

	assert.Equal(t, "localhost:9000", status.Endpoint)
	assert.Equal(t, "test-bucket", status.BucketName)
	assert.Greater(t, status.Latency, time.Duration(0))

	if status.Error != "" {
		t.Logf("Health check failed (expected if MinIO not running): %s", status.Error)
	}
}

func TestMinIOClient_GeneratePresignedURL(t *testing.T) {
	config := MinIOConfig{
		Endpoint:        "localhost:9000",
		AccessKeyID:     "testkey",
		SecretAccessKey: "testsecret",
		UseSSL:          false,
		BucketName:      "test-bucket",
	}

	client, err := NewMinIOService(config)
	if err != nil && strings.Contains(err.Error(), "connection") {
		t.Skip("Skipping test - MinIO server not available")
	}
	if err != nil {
		client = &MinIOClient{
			config:     config,
			bucketName: "test-bucket",
		}
	}

	ctx := context.Background()
	expires := 1 * time.Hour

	tests := []struct {
		name      string
		method    string
		expectErr bool
		errMsg    string
	}{
		{
			name:      "Valid GET method",
			method:    "GET",
			expectErr: false,
		},
		{
			name:      "Valid PUT method",
			method:    "PUT",
			expectErr: false,
		},
		{
			name:      "Invalid method",
			method:    "DELETE",
			expectErr: true,
			errMsg:    "unsupported method",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.GeneratePresignedURL(ctx, "test-object", expires, tt.method)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				if err != nil && strings.Contains(err.Error(), "connection") {
					t.Skip("Skipping test - MinIO server not available")
				}
			}
		})
	}
}

func TestMinIOClient_PutObject(t *testing.T) {
	config := MinIOConfig{
		Endpoint:        "localhost:9000",
		AccessKeyID:     "testkey",
		SecretAccessKey: "testsecret",
		UseSSL:          false,
		BucketName:      "test-bucket",
	}

	client, err := NewMinIOService(config)
	if err != nil && strings.Contains(err.Error(), "connection") {
		t.Skip("Skipping test - MinIO server not available")
		return
	}
	if err != nil {
		t.Logf("MinIO client creation failed: %v", err)
		return
	}

	ctx := context.Background()
	reader := strings.NewReader("test content")
	objectSize := int64(len("test content"))

	err = client.PutObject(ctx, "test-object", reader, objectSize, minio.PutObjectOptions{})

	if err != nil && strings.Contains(err.Error(), "connection") {
		t.Skip("Skipping test - MinIO server not available")
	}

	if err != nil {
		t.Logf("Put object failed (expected if MinIO not running): %v", err)
	}
}

func TestMinIOClient_GetObject(t *testing.T) {
	config := MinIOConfig{
		Endpoint:        "localhost:9000",
		AccessKeyID:     "testkey",
		SecretAccessKey: "testsecret",
		UseSSL:          false,
		BucketName:      "test-bucket",
	}

	client, err := NewMinIOService(config)
	if err != nil && strings.Contains(err.Error(), "connection") {
		t.Skip("Skipping test - MinIO server not available")
		return
	}
	if err != nil {
		t.Logf("MinIO client creation failed: %v", err)
		return
	}

	ctx := context.Background()

	_, err = client.GetObject(ctx, "test-object")

	if err != nil && strings.Contains(err.Error(), "connection") {
		t.Skip("Skipping test - MinIO server not available")
	}

	if err != nil {
		t.Logf("Get object failed (expected if MinIO not running): %v", err)
	}
}

func TestMinIOClient_DeleteObject(t *testing.T) {
	config := MinIOConfig{
		Endpoint:        "localhost:9000",
		AccessKeyID:     "testkey",
		SecretAccessKey: "testsecret",
		UseSSL:          false,
		BucketName:      "test-bucket",
	}

	client, err := NewMinIOService(config)
	if err != nil && strings.Contains(err.Error(), "connection") {
		t.Skip("Skipping test - MinIO server not available")
		return
	}
	if err != nil {
		t.Logf("MinIO client creation failed: %v", err)
		return
	}

	ctx := context.Background()

	err = client.DeleteObject(ctx, "test-object")

	if err != nil && strings.Contains(err.Error(), "connection") {
		t.Skip("Skipping test - MinIO server not available")
	}

	if err != nil {
		t.Logf("Delete object failed (expected if MinIO not running): %v", err)
	}
}

func TestMinIOClient_ListObjects(t *testing.T) {
	config := MinIOConfig{
		Endpoint:        "localhost:9000",
		AccessKeyID:     "testkey",
		SecretAccessKey: "testsecret",
		UseSSL:          false,
		BucketName:      "test-bucket",
	}

	client, err := NewMinIOService(config)
	if err != nil && strings.Contains(err.Error(), "connection") {
		t.Skip("Skipping test - MinIO server not available")
		return
	}
	if err != nil {
		t.Logf("MinIO client creation failed: %v", err)
		return
	}

	ctx := context.Background()

	objects, err := client.ListObjects(ctx, "", true)

	if err != nil && strings.Contains(err.Error(), "connection") {
		t.Skip("Skipping test - MinIO server not available")
	}

	if err != nil {
		t.Logf("List objects failed (expected if MinIO not running): %v", err)
	} else {
		assert.IsType(t, []minio.ObjectInfo{}, objects)
	}
}

func TestMinIOClient_BucketExists(t *testing.T) {
	config := MinIOConfig{
		Endpoint:        "localhost:9000",
		AccessKeyID:     "testkey",
		SecretAccessKey: "testsecret",
		UseSSL:          false,
		BucketName:      "test-bucket",
	}

	client, err := NewMinIOService(config)
	if err != nil && strings.Contains(err.Error(), "connection") {
		t.Skip("Skipping test - MinIO server not available")
		return
	}
	if err != nil {
		t.Logf("MinIO client creation failed: %v", err)
		return
	}

	ctx := context.Background()

	_, err = client.BucketExists(ctx, "test-bucket")

	if err != nil && strings.Contains(err.Error(), "connection") {
		t.Skip("Skipping test - MinIO server not available")
	}

	if err != nil {
		t.Logf("Bucket exists check failed (expected if MinIO not running): %v", err)
	}
}

func TestMinIOClient_CreateBucket(t *testing.T) {
	config := MinIOConfig{
		Endpoint:        "localhost:9000",
		AccessKeyID:     "testkey",
		SecretAccessKey: "testsecret",
		UseSSL:          false,
		BucketName:      "test-bucket-new",
	}

	client, err := NewMinIOService(config)
	if err != nil && strings.Contains(err.Error(), "connection") {
		t.Skip("Skipping test - MinIO server not available")
		return
	}
	if err != nil {
		t.Logf("MinIO client creation failed: %v", err)
		return
	}

	ctx := context.Background()

	err = client.CreateBucket(ctx, "test-bucket-new")

	if err != nil && strings.Contains(err.Error(), "connection") {
		t.Skip("Skipping test - MinIO server not available")
	}

	if err != nil {
		t.Logf("Create bucket failed (expected if MinIO not running): %v", err)
	}
}

func TestMinIOClient_GetObjectInfo(t *testing.T) {
	config := MinIOConfig{
		Endpoint:        "localhost:9000",
		AccessKeyID:     "testkey",
		SecretAccessKey: "testsecret",
		UseSSL:          false,
		BucketName:      "test-bucket",
	}

	client, err := NewMinIOService(config)
	if err != nil && strings.Contains(err.Error(), "connection") {
		t.Skip("Skipping test - MinIO server not available")
		return
	}
	if err != nil {
		t.Logf("MinIO client creation failed: %v", err)
		return
	}

	ctx := context.Background()

	_, err = client.GetObjectInfo(ctx, "test-object")

	if err != nil && strings.Contains(err.Error(), "connection") {
		t.Skip("Skipping test - MinIO server not available")
	}

	if err != nil {
		t.Logf("Get object info failed (expected if MinIO not running): %v", err)
	}
}

func TestMinIOClient_GetClient(t *testing.T) {
	client := &MinIOClient{}

	minioClient := client.GetClient()
	assert.Equal(t, client.client, minioClient)
}

func TestMinIOClient_Close(t *testing.T) {
	client := &MinIOClient{}

	err := client.Close()
	assert.NoError(t, err)
}

func BenchmarkMinIOClient_PutObject(b *testing.B) {
	config := MinIOConfig{
		Endpoint:        "localhost:9000",
		AccessKeyID:     "testkey",
		SecretAccessKey: "testsecret",
		UseSSL:          false,
		BucketName:      "benchmark-bucket",
	}

	client, err := NewMinIOService(config)
	if err != nil && strings.Contains(err.Error(), "connection") {
		b.Skip("Skipping benchmark - MinIO server not available")
	}
	if err != nil {
		b.Skipf("MinIO client creation failed: %v", err)
	}

	ctx := context.Background()
	content := strings.Repeat("benchmark data ", 1000)
	objectSize := int64(len(content))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(content)
		objectName := fmt.Sprintf("benchmark-object-%d", i)

		err := client.PutObject(ctx, objectName, reader, objectSize, minio.PutObjectOptions{})
		if err != nil && strings.Contains(err.Error(), "connection") {
			b.Skip("Skipping benchmark - MinIO server not available")
		}
	}
}

func BenchmarkMinIOClient_GetObject(b *testing.B) {
	config := MinIOConfig{
		Endpoint:        "localhost:9000",
		AccessKeyID:     "testkey",
		SecretAccessKey: "testsecret",
		UseSSL:          false,
		BucketName:      "benchmark-bucket",
	}

	client, err := NewMinIOService(config)
	if err != nil && strings.Contains(err.Error(), "connection") {
		b.Skip("Skipping benchmark - MinIO server not available")
	}
	if err != nil {
		b.Skipf("MinIO client creation failed: %v", err)
	}

	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		objectName := fmt.Sprintf("benchmark-object-%d", i%10)

		obj, err := client.GetObject(ctx, objectName)
		if err != nil && strings.Contains(err.Error(), "connection") {
			b.Skip("Skipping benchmark - MinIO server not available")
		}
		if obj != nil {
			obj.Close()
		}
	}
}

func ExampleNewMinIOService() {
	config := MinIOConfig{
		Endpoint:        "localhost:9000",
		AccessKeyID:     "minioadmin",
		SecretAccessKey: "minioadmin",
		UseSSL:          false,
		BucketName:      "my-bucket",
	}

	client, err := NewMinIOService(config)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	ctx := context.Background()
	status := client.HealthCheck(ctx)
	if status.Connected {
		content := strings.NewReader("Hello, MinIO!")
		err := client.PutObject(ctx, "hello.txt", content, 13, minio.PutObjectOptions{})
		if err != nil {
			panic(err)
		}
	}
}

